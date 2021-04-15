package article

type Article struct {
	ID          string `json:"id" xlsx:"0"`
	DownloadURL string `json:"downloadUrl" xlsx:"1"`
	CountWBM    int    `json:"wcount" xlsx:"2"`
	CountGoogle int    `json:"gcount" xlsx:"3"`
}
