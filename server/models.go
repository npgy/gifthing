package main

type SetGifRequest struct {
	GifURL  string `json:"gifUrl"`
	IsTenor bool   `json:"isTenor"`
}

type TenorMp4Request struct {
	TenorGifURL string `json:"tenorGifUrl"`
}

type MediaFormat struct {
	URL      string  `json:"url"`
	Duration float64 `json:"duration"`
	Preview  string  `json:"preview"`
	Dims     []int   `json:"dims"`
	Size     int     `json:"size"`
}

type MediaFormats struct {
	MP4  MediaFormat `json:"mp4"`
	WebP MediaFormat `json:"webp"`
}

type TenorPost struct {
	ID           string       `json:"id"`
	MediaFormats MediaFormats `json:"media_formats"`
}

type TenorResponse struct {
	Results []TenorPost `json:"results"`
}
