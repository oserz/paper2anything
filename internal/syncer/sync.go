package syncer

import (
	"encoding/json"
	"fmt"
	"os"
	"paperless2anythingllm/internal/anything"
	"paperless2anythingllm/internal/config"
	"paperless2anythingllm/internal/paperless"
	"paperless2anythingllm/internal/util"
	"path/filepath"
	"strings"
)

type state struct {
	Docs map[int]stateDoc `json:"docs"`
}

type stateDoc struct {
	Workspaces map[string]string `json:"workspaces,omitempty"`
	DocURL     string            `json:"doc_url,omitempty"`
	Modified   string            `json:"modified"`
}

func (d stateDoc) workspaceMap() map[string]string {
	if len(d.Workspaces) > 0 {
		m := make(map[string]string, len(d.Workspaces))
		for k, v := range d.Workspaces {
			m[k] = v
		}
		return m
	}
	m := map[string]string{}
	return m
}

func sameWorkspaceSet(prev map[string]string, current []string) bool {
	if len(prev) != len(current) {
		return false
	}
	for _, s := range current {
		if _, ok := prev[s]; !ok {
			return false
		}
	}
	return true
}

func loadState(path string) (*state, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return &state{Docs: map[int]stateDoc{}}, nil
	}
	var s state
	if err := json.Unmarshal(b, &s); err != nil {
		return &state{Docs: map[int]stateDoc{}}, nil
	}
	if s.Docs == nil {
		s.Docs = map[int]stateDoc{}
	}
	return &s, nil
}

func saveState(path string, s *state) error {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path, b, 0644)
}

func collectAllDocNames(st *state) []string {
	if st == nil {
		return nil
	}
	namesSet := map[string]struct{}{}
	for _, doc := range st.Docs {
		for _, url := range doc.workspaceMap() {
			if url == "" {
				continue
			}
			namesSet[url] = struct{}{}
		}
	}
	var names []string
	for n := range namesSet {
		names = append(names, n)
	}
	return names
}

func Run(cfg config.Config, dryRun bool) error {
	pc := paperless.New(cfg.Paperless.BaseURL, cfg.Paperless.Token, cfg.Paperless.PageSize)
	ac := anything.New(cfg.AnythingLLM.BaseURL, cfg.AnythingLLM.APIKey)
	docs, err := pc.ListDocuments()
	if err != nil {
		fmt.Printf("Failed to list documents from Paperless: %v\n", err)
		return err
	}
	st, err := loadState(cfg.Sync.StateFile)
	if err != nil {
		fmt.Printf("Failed to load sync state: %v\n", err)
		return err
	}
	deleteNames := map[string]struct{}{}
	if dryRun {
		fmt.Printf("Starting dry-run sync, total documents: %d\n", len(docs))
	} else {
		fmt.Printf("Starting sync, total documents: %d\n", len(docs))
	}
	present := map[int]struct{}{}
	for _, d := range docs {
		fmt.Printf("Processing document ID %d: %s\n", d.ID, d.Title)
		present[d.ID] = struct{}{}
		groups := paperless.GroupKeys(d, cfg.Sync.Grouping, cfg.Sync.DefaultWorkspace)
		workspaces := map[string]string{}
		for _, group := range groups {
			if group == "" {
				continue
			}
			slug := util.Slugify(group)
			if !dryRun {
				var slugnew string
				if slugnew, err = ac.EnsureWorkspace(group, slug); err != nil {
					fmt.Printf("Failed to ensure workspace %s: %v\n", group, err)
					return err
				}
				slug = slugnew
			}
			workspaces[slug] = group
		}
		if len(workspaces) == 0 {
			continue
		}
		var currentSlugs []string
		for slug := range workspaces {
			currentSlugs = append(currentSlugs, slug)
		}
		prev, ok := st.Docs[d.ID]
		prevMap := prev.workspaceMap()
		mod := d.Modified
		workspaceChanged := !sameWorkspaceSet(prevMap, currentSlugs)
		changed := !ok || prev.Modified != mod || workspaceChanged
		if !changed {
			continue
		}
		fmt.Printf("Document ID %d changed, isModified: %v, workspaceChanged: %v\n", d.ID, prev.Modified != mod, workspaceChanged)
		if dryRun {
			fmt.Printf("Would update document %d in workspaces: %s\n", d.ID, strings.Join(currentSlugs, ", "))
			continue
		}
		prevSlugs := map[string]struct{}{}
		for slug := range prevMap {
			prevSlugs[slug] = struct{}{}
		}
		toAdd := []string{}
		toUpdate := []string{}
		for _, slug := range currentSlugs {
			if _, exists := prevSlugs[slug]; !exists {
				toAdd = append(toAdd, slug)
			} else if prev.Modified != mod {
				toUpdate = append(toUpdate, slug)
			}
		}
		toRemove := []string{}
		for slug := range prevSlugs {
			found := false
			for _, s := range currentSlugs {
				if s == slug {
					found = true
					break
				}
			}
			if !found {
				toRemove = append(toRemove, slug)
			}
		}
		fmt.Printf("To add: %v, to update: %v, to remove: %v\n", toAdd, toUpdate, toRemove)
		newDocURLs := map[string]string{}
		uploadedDocs := []string{}
		if len(toAdd) > 0 || len(toUpdate) > 0 {
			fp, err := pc.Download(d)
			if err != nil {
				return err
			}
			for _, slug := range append(toAdd, toUpdate...) {
				docURL, err := ac.UploadDocument(fp, slug)
				if err != nil {
					return err
				}
				newDocURLs[slug] = docURL
				uploadedDocs = append(uploadedDocs, docURL)
			}
		}
		rollback := func() {
			fmt.Printf("Rolling back uploaded documents for document ID %d\n", d.ID)
			if len(uploadedDocs) > 0 {
				_ = ac.RemoveDocuments(uploadedDocs)
			}
		}
		for _, slug := range currentSlugs {
			adds := []string{}
			removes := []string{}
			if docURL, ok := newDocURLs[slug]; ok {
				adds = append(adds, docURL)
				if old, ok := prevMap[slug]; ok && old != "" {
					removes = append(removes, old)
					deleteNames[old] = struct{}{}
				}
			}
			if len(adds) == 0 && len(removes) == 0 {
				continue
			}
			if err := ac.UpdateEmbeddings(slug, adds, removes); err != nil {
				rollback()
				return err
			}
		}
		for _, slug := range toRemove {
			if old, ok := prevMap[slug]; ok && old != "" {
				if err := ac.UpdateEmbeddings(slug, []string{}, []string{old}); err != nil {
					return err
				}
				deleteNames[old] = struct{}{}
			}
		}
		nextMap := map[string]string{}
		for _, slug := range currentSlugs {
			if docURL, ok := newDocURLs[slug]; ok {
				nextMap[slug] = docURL
			} else if old, ok := prevMap[slug]; ok && old != "" {
				nextMap[slug] = old
			}
		}
		st.Docs[d.ID] = stateDoc{Workspaces: nextMap, Modified: mod}
	}
	if dryRun {
		for id, prev := range st.Docs {
			if _, ok := present[id]; ok {
				continue
			}
			prevMap := prev.workspaceMap()
			var slugs []string
			for slug := range prevMap {
				slugs = append(slugs, slug)
			}
			if len(slugs) == 0 {
				continue
			}
			fmt.Printf("Would remove document %d from workspaces: %s\n", id, strings.Join(slugs, ", "))
		}
		return nil
	}
	if len(deleteNames) > 0 {
		names := make([]string, 0, len(deleteNames))
		for n := range deleteNames {
			names = append(names, n)
		}
		if err := ac.RemoveDocuments(names); err != nil {
			return err
		}
		fmt.Printf("Removed %d documents from AnythingLLM.\n", len(names))
	}
	for id, prev := range st.Docs {
		if _, ok := present[id]; ok {
			continue
		}
		prevMap := prev.workspaceMap()
		for slug, docURL := range prevMap {
			if docURL != "" && slug != "" {
				adds := []string{}
				if err := ac.UpdateEmbeddings(slug, adds, []string{docURL}); err != nil {
					return err
				}
				fmt.Printf("Removed Embeddings for document %d from workspace %s.\n", id, slug)
			}
		}
		delete(st.Docs, id)
	}
	if !dryRun {
		if err := saveState(cfg.Sync.StateFile, st); err != nil {
			return err
		}
	}
	return nil
}

func ClearAnything(cfg config.Config, dryRun bool) error {
	ac := anything.New(cfg.AnythingLLM.BaseURL, cfg.AnythingLLM.APIKey)
	st, err := loadState(cfg.Sync.StateFile)
	if err != nil {
		return err
	}
	names := collectAllDocNames(st)
	slugs, err := ac.ListWorkspaceSlugs()
	if err != nil {
		return err
	}
	if dryRun {
		fmt.Printf("Would delete %d documents from AnythingLLM.\n", len(names))
		fmt.Printf("Would delete %d workspaces from AnythingLLM.\n", len(slugs))
		return nil
	}
	if len(names) > 0 {
		if err := ac.RemoveDocuments(names); err != nil {
			return err
		}
	}
	for _, slug := range slugs {
		if err := ac.DeleteWorkspace(slug); err != nil {
			return err
		}
	}
	st.Docs = map[int]stateDoc{}
	if err := saveState(cfg.Sync.StateFile, st); err != nil {
		return err
	}
	fmt.Printf("Deleted %d documents and %d workspaces from AnythingLLM and reset sync state.\n", len(names), len(slugs))
	return nil
}
