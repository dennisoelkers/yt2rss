package feed

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/dnhn/yt2rss/internal/models"
)

type Generator struct {
	BaseURL string
}

func NewGenerator(baseURL string) *Generator {
	return &Generator{BaseURL: baseURL}
}

func (g *Generator) Generate(feed *models.Feed, videos []models.Video) ([]byte, error) {
	return g.GenerateWithBaseURL(feed, videos, g.BaseURL)
}

func (g *Generator) GenerateWithBaseURL(feed *models.Feed, videos []models.Video, baseURL string) ([]byte, error) {
	rss := RSS{
		XMLNsItunes: "http://www.itunes.com/dtds/podcast-1.0.dtd",
		XMLNsMedia:  "http://search.yahoo.com/mrss/",
		XMLNsAtom:   "http://www.w3.org/2005/Atom",
		Channel: RSSChannel{
			Title:          feed.Title,
			Link:           feed.YoutubeURL,
			Description:    feed.Description,
			Language:       "en",
			ItunesAuthor:   feed.Title,
			ItunesSummary:  feed.Description,
			ItunesCategory: ItunesCategory{Text: "Technology"},
			AtomLink: AtomLink{
				Rel:  "self",
				Href: fmt.Sprintf("%s/rss/%s", baseURL, feed.ID),
				Type: "application/rss+xml",
			},
		},
	}

	for _, video := range videos {
		item := RSSItem{
			Title:          video.Title,
			Link:           fmt.Sprintf("https://www.youtube.com/watch?v=%s", video.YoutubeID),
			Guid:           Guid{Value: video.YoutubeID, IsPermaLink: false},
			PubDate:        video.PublishedAt.Format(time.RFC1123),
			Description:    video.Description,
			ItunesTitle:    video.Title,
			ItunesDuration: formatDuration(video.Duration),
			Enclosure: Enclosure{
				URL:    fmt.Sprintf("%s/stream/%s/%s", baseURL, feed.ID, video.YoutubeID),
				Type:   "audio/mp4",
				Length: "0",
			},
			MediaContent: MediaContent{
				URL:      fmt.Sprintf("%s/stream/%s/%s", baseURL, feed.ID, video.YoutubeID),
				Medium:   "audio",
				Duration: video.Duration,
				MediaThumbnail: MediaThumbnail{
					URL: video.ThumbnailURL,
				},
			},
		}
		rss.Channel.Items = append(rss.Channel.Items, item)
	}

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(rss); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

type RSS struct {
	XML         xml.Name   `xml:"rss"`
	XMLNsItunes string     `xml:"xmlns:itunes,attr"`
	XMLNsMedia  string     `xml:"xmlns:media,attr"`
	XMLNsAtom   string     `xml:"xmlns:atom,attr"`
	Version     string     `xml:"version,attr"`
	Channel     RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	XMLName        xml.Name       `xml:"channel"`
	Title          string         `xml:"title"`
	Link           string         `xml:"link"`
	Description    string         `xml:"description"`
	Language       string         `xml:"language"`
	ItunesAuthor   string         `xml:"itunes:author"`
	ItunesSummary  string         `xml:"itunes:summary"`
	ItunesCategory ItunesCategory `xml:"itunes:category"`
	AtomLink       AtomLink       `xml:"atom:link"`
	Items          []RSSItem      `xml:"item"`
}

type ItunesCategory struct {
	XMLName xml.Name `xml:"itunes:category"`
	Text    string   `xml:"text,attr"`
}

type AtomLink struct {
	XMLName xml.Name `xml:"atom:link"`
	Rel     string   `xml:"rel,attr"`
	Href    string   `xml:"href,attr"`
	Type    string   `xml:"type,attr"`
}

type RSSItem struct {
	XMLName        xml.Name     `xml:"item"`
	Title          string       `xml:"title"`
	Link           string       `xml:"link"`
	Guid           Guid         `xml:"guid"`
	PubDate        string       `xml:"pubDate"`
	Description    string       `xml:"description"`
	ItunesTitle    string       `xml:"itunes:title"`
	ItunesDuration string       `xml:"itunes:duration"`
	Enclosure      Enclosure    `xml:"enclosure"`
	MediaContent   MediaContent `xml:"media:content"`
}

type Guid struct {
	XMLName     xml.Name `xml:"guid"`
	Value       string   `xml:",chardata"`
	IsPermaLink bool     `xml:"isPermaLink,attr"`
}

type Enclosure struct {
	XMLName xml.Name `xml:"enclosure"`
	URL     string   `xml:"url,attr"`
	Type    string   `xml:"type,attr"`
	Length  string   `xml:"length,attr"`
}

type MediaContent struct {
	XMLName        xml.Name       `xml:"media:content"`
	URL            string         `xml:"url,attr"`
	Medium         string         `xml:"medium,attr"`
	Duration       int            `xml:"duration,attr"`
	MediaThumbnail MediaThumbnail `xml:"media:thumbnail"`
}

type MediaThumbnail struct {
	XMLName xml.Name `xml:"media:thumbnail"`
	URL     string   `xml:"url,attr"`
}
