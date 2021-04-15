package article

// Article describes article metadata to be written to the spreadsheet file.
type Article struct {
	ID          string `json:"id" xlsx:"0"`
	DownloadURL string `json:"downloadUrl" xlsx:"1"`
}
