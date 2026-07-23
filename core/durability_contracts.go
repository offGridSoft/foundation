package core

const (
	DurableStagePattern    = ".foundation-stage-*"
	DurableCopyBufferBytes = 32 << 10

	ErrFmtDurableAppendRequest = "durability.AppendRequest: %w"
	ErrFmtDurableAppendOpen    = "open durable append target: %w"
	ErrFmtDurableAppendInspect = "inspect durable append target: %w"
	ErrFmtDurableAppendWrite   = "write durable append target: %w"
	ErrFmtDurableAppendSync    = "sync durable append target: %w"
	ErrFmtDurableAppendClose   = "close durable append target: %w"
)
