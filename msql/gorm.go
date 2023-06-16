package msql

func NewGormDB(opts ...Option) {
	cfg := DefaultDBOption()
	for _, opt := range opts {
		opt(cfg)
	}
	//_db, _ := NewDB(opts...)
	//gorm.Open(, &gorm.Config{})
}
