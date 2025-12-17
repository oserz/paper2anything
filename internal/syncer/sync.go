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
)

type state struct {
	Docs map[int]stateDoc `json:"docs"`
}

type stateDoc struct {
	Workspace string `json:"workspace"`
	DocURL    string `json:"doc_url"`
	Modified  string `json:"modified"`
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

func Run(cfg config.Config, dryRun bool) error {
	pc := paperless.New(cfg.Paperless.BaseURL, cfg.Paperless.Token, cfg.Paperless.PageSize)
	ac := anything.New(cfg.AnythingLLM.BaseURL, cfg.AnythingLLM.APIKey)
	docs, err := pc.ListDocuments()
	if err != nil {
		return err
	}
	st, err := loadState(cfg.Sync.StateFile)
	if err != nil {
		return err
	}
	present := map[int]struct{}{}
	for _, d := range docs {
		present[d.ID] = struct{}{}
		group := paperless.GroupKey(d, cfg.Sync.Grouping, cfg.Sync.DefaultWorkspace)
		slug := util.Slugify(group)
		if !dryRun {
			var slugnew string
			if slugnew, err = ac.EnsureWorkspace(group, slug); err != nil {
				return err
			}
			slug = slugnew
		}
		prev, ok := st.Docs[d.ID]
		mod := d.Modified
		changed := !ok || prev.Modified != mod || prev.Workspace != slug
		if !changed {
			continue
		}
		if dryRun {
			fmt.Printf("将更新文档 %d 至工作区 %s\n", d.ID, slug)
			continue
		}
		fp, err := pc.Download(d)
		if err != nil {
			return err
		}
		docURL, err := ac.UploadDocument(fp, slug)
		if err != nil {
			return err
		}
		adds := []string{docURL}

		rollback := func() {
			_ = ac.RemoveDocuments([]string{docURL})
		}
		removes := []string{}
		if ok && prev.DocURL != "" && prev.Workspace == slug {
			removes = append(removes, prev.DocURL)
		}
		if prev.Workspace != "" && prev.Workspace != slug && prev.DocURL != "" {
			if err := ac.UpdateEmbeddings(prev.Workspace, nil, []string{prev.DocURL}); err != nil {
				rollback()
				return err
			}
		}
		if err := ac.UpdateEmbeddings(slug, adds, removes); err != nil {
			rollback()
			return err
		}
		st.Docs[d.ID] = stateDoc{Workspace: slug, DocURL: docURL, Modified: mod}
	}
	for id, prev := range st.Docs {
		if _, ok := present[id]; ok {
			continue
		}
		if dryRun {
			fmt.Printf("将从工作区 %s 移除文档 %d\n", prev.Workspace, id)
			continue
		}
		if prev.DocURL != "" && prev.Workspace != "" {
			_ = ac.UpdateEmbeddings(prev.Workspace, nil, []string{prev.DocURL})
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
