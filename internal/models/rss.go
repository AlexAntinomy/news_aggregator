package models

import "encoding/xml"

// RSS представляет корневой элемент RSS-документа.
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Channel содержит заголовок и список элементов Item.
type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

// Item представляет одну публикацию из RSS-ленты.
// Поле FeedID заполняется локально, при отправке в канал, и не участвует в XML-декодировании.
type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Link        string `xml:"link"`
	FeedID      int    `xml:"-"`
}
