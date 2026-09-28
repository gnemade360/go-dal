package file

type Option func(*FileProvider)

func WithDir(dir string) Option {
	return func(p *FileProvider) {
		p.dir = dir
	}
}

func WithFormat(f Format) Option {
	return func(p *FileProvider) {
		p.format = f
	}
}

func WithPrettyPrint(pretty bool) Option {
	return func(p *FileProvider) {
		p.pretty = pretty
	}
}
