package custody

const (
	ErrFmtULID            = "custody.ULID: %w"
	ErrFmtRelease         = "custody.ReleaseIdentity: %w"
	ErrFmtArtifactName    = "custody.ArtifactName: %w"
	ErrFmtArtifact        = "custody.ArtifactDescriptor: %w"
	ErrFmtObjectPath      = "custody.ObjectPath: %w"
	ErrFmtGeneration      = "custody.Generation: %w"
	ErrFmtStorage         = "custody.Storage: %w"
	ErrFmtRetention       = "custody.Retention: %w"
	ErrFmtOpenRequest     = "custody.SessionOpenRequest: %w"
	ErrFmtOpenResponse    = "custody.SessionOpenResponse: %w"
	ErrFmtUploadedObject  = "custody.UploadedObject: %w"
	ErrFmtFinalize        = "custody.FinalizeRequest: %w"
	ErrFmtReceipt         = "custody.ReceiptBody: %w"
	ErrFmtOpenDisposition = "custody.SessionOpenDisposition: %w"
	ErrFmtTimestamp       = "custody.TimestampProof: %w"
)
