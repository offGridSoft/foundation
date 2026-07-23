package core

const (
	DurableStagePrefix         = ".foundation-stage-"
	DurableStagePattern        = DurableStagePrefix + "*"
	DurableStageTokenBytes     = 16
	DurableStageCreateAttempts = 32
	DurableCopyBufferBytes     = 32 << 10

	ErrFmtDurableAppendRequest = "durability.AppendRequest: %w"
	ErrFmtDurableAppendOpen    = "open durable append target: %w"
	ErrFmtDurableAppendInspect = "inspect durable append target: %w"
	ErrFmtDurableAppendWrite   = "write durable append target: %w"
	ErrFmtDurableAppendSync    = "sync durable append target: %w"
	ErrFmtDurableAppendClose   = "close durable append target: %w"
)
