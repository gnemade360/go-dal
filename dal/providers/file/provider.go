package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gnemade360/go-dal/dal/errors"
	"github.com/gnemade360/go-dal/dal/interfaces"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

type FileProvider struct {
	dir    string
	format Format
	pretty bool
	mu     sync.RWMutex
}

func New(opts ...Option) *FileProvider {
	p := &FileProvider{
		dir:    "./data",
		format: FormatJSON,
		pretty: true,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *FileProvider) FindByID(ctx context.Context, tableName string, id string, dest interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	path := p.filePath(tableName, id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NewNotFoundError(tableName, id)
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	return p.unmarshal(data, dest)
}

func (p *FileProvider) FindAll(ctx context.Context, tableName string, dest interface{}, opts interfaces.QueryOptions) error {
	return p.FindByFilter(ctx, tableName, dest, interfaces.Filter{}, opts)
}

func (p *FileProvider) FindByFilter(ctx context.Context, tableName string, dest interface{}, filter interfaces.Filter, opts interfaces.QueryOptions) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	dir := p.tableDir(tableName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return setSliceEmpty(dest)
		}
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	var records []map[string]interface{}
	ext := "." + string(p.format)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ext) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var record map[string]interface{}
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		if matchesFilter(record, filter) {
			records = append(records, record)
		}
	}

	sortRecords(records, opts.OrderBy)

	if opts.Offset > 0 && opts.Offset < len(records) {
		records = records[opts.Offset:]
	} else if opts.Offset >= len(records) {
		records = nil
	}

	if opts.Limit > 0 && opts.Limit < len(records) {
		records = records[:opts.Limit]
	}

	return unmarshalRecords(records, dest)
}

func (p *FileProvider) Count(ctx context.Context, tableName string, filter interfaces.Filter) (int64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	dir := p.tableDir(tableName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read dir %s: %w", dir, err)
	}

	ext := "." + string(p.format)
	var count int64

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ext) {
			continue
		}

		if isEmptyFilter(filter) {
			count++
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var record map[string]interface{}
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		if matchesFilter(record, filter) {
			count++
		}
	}

	return count, nil
}

func (p *FileProvider) Create(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	dir := p.tableDir(tableName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	path := p.filePath(tableName, id)
	if _, err := os.Stat(path); err == nil {
		return errors.NewDuplicateError(tableName, "id", id)
	}

	content, err := p.marshal(data)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return os.WriteFile(path, content, 0644)
}

func (p *FileProvider) Update(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	path := p.filePath(tableName, id)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.NewNotFoundError(tableName, id)
		}
		return err
	}

	content, err := p.marshal(data)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return os.WriteFile(path, content, 0644)
}

func (p *FileProvider) Delete(ctx context.Context, tableName string, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	path := p.filePath(tableName, id)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.NewNotFoundError(tableName, id)
		}
		return err
	}

	return os.Remove(path)
}

func (p *FileProvider) BeginTx(ctx context.Context) (interfaces.StoreTx, error) {
	return &fileTx{provider: p, ops: nil, committed: false}, nil
}

func (p *FileProvider) tableDir(tableName string) string {
	return filepath.Join(p.dir, tableName)
}

func (p *FileProvider) filePath(tableName, id string) string {
	return filepath.Join(p.dir, tableName, id+"."+string(p.format))
}

func (p *FileProvider) marshal(data interface{}) ([]byte, error) {
	if p.pretty {
		return json.MarshalIndent(data, "", "  ")
	}
	return json.Marshal(data)
}

func (p *FileProvider) unmarshal(data []byte, dest interface{}) error {
	return json.Unmarshal(data, dest)
}

func sortRecords(records []map[string]interface{}, orderBy []interfaces.OrderClause) {
	if len(orderBy) == 0 {
		return
	}

	sort.SliceStable(records, func(i, j int) bool {
		for _, clause := range orderBy {
			vi := records[i][clause.Field]
			vj := records[j][clause.Field]

			cmp := compareValues(vi, vj)
			if cmp == 0 {
				continue
			}

			if strings.ToUpper(clause.Direction) == "DESC" {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

func compareValues(a, b interface{}) int {
	sa := fmt.Sprintf("%v", a)
	sb := fmt.Sprintf("%v", b)

	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}

func setSliceEmpty(dest interface{}) error {
	return unmarshalRecords(nil, dest)
}

func unmarshalRecords(records []map[string]interface{}, dest interface{}) error {
	if records == nil {
		records = []map[string]interface{}{}
	}
	data, err := json.Marshal(records)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}
