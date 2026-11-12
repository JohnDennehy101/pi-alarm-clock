package sonos

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

type RadioStation struct {
	Title    string
	URI      string
	Metadata string
}

type didlLite struct {
	Items []didlItem `xml:"item"`
}

type didlItem struct {
	Title       string         `xml:"title"`
	Resources   []didlResource `xml:"res"`
	ResMD       string         `xml:"resMD"`
	Type        string         `xml:"type"`
	Description string         `xml:"description"`
}

type didlResource struct {
	URI          string `xml:",chardata"`
	ProtocolInfo string `xml:"protocolInfo,attr"`
}

func (item *didlItem) UnmarshalXML(
	decoder *xml.Decoder,
	start xml.StartElement,
) error {
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}

		switch element := token.(type) {
		case xml.StartElement:
			if err := item.handleElement(decoder, element); err != nil {
				return err
			}

		case xml.EndElement:
			if element.Name == start.Name {
				return nil
			}
		}
	}
}

func (item *didlItem) handleElement(decoder *xml.Decoder, element xml.StartElement) error {
	switch element.Name.Local {
	case "title":
		return decoder.DecodeElement(&item.Title, &element)

	case "res":
		resource, err := parseResource(decoder, element)
		if err != nil {
			return err
		}
		item.Resources = append(item.Resources, resource)
		return nil

	case "resMD":
		return decoder.DecodeElement(&item.ResMD, &element)

	default:
		return decoder.Skip()
	}
}

func parseResource(decoder *xml.Decoder, element xml.StartElement) (didlResource, error) {
	var resource didlResource

	for _, attr := range element.Attr {
		if attr.Name.Local == "protocolInfo" {
			resource.ProtocolInfo = attr.Value
		}
	}

	if err := decoder.DecodeElement(&resource.URI, &element); err != nil {
		return didlResource{}, err
	}

	resource.URI = strings.TrimSpace(resource.URI)
	return resource, nil
}

func (c *Client) PlayRadioStation(ctx context.Context, name string) error {
	stations, err := c.GetRadioStations(ctx)
	if err != nil {
		return err
	}

	for _, station := range stations {
		if strings.EqualFold(station.Title, name) {
			return c.PlayURI(ctx, station.URI, station.Metadata)
		}
	}

	return fmt.Errorf("radio station %q not found in Sonos favourites", name)
}

func (c *Client) GetRadioStations(
	ctx context.Context,
) ([]RadioStation, error) {

	result, _, _, _, err := c.contentDir.BrowseCtx(
		ctx,
		"FV:2",
		"BrowseDirectChildren",
		"*",
		0,
		100,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("browse Sonos favourites: %w", err)
	}

	var didl didlLite

	if err := xml.Unmarshal([]byte(result), &didl); err != nil {
		return nil, fmt.Errorf("parse Sonos favourites: %w", err)
	}

	var stations []RadioStation

	for _, item := range didl.Items {

		if len(item.Resources) == 0 {
			continue
		}

		resource := item.Resources[0]

		station := RadioStation{
			Title:    item.Title,
			URI:      resource.URI,
			Metadata: strings.TrimSpace(item.ResMD),
		}

		stations = append(stations, station)
	}

	return stations, nil
}

func xmlEscape(value string) string {
	var builder strings.Builder
	_ = xml.EscapeText(&builder, []byte(value))
	return builder.String()
}

func (c *Client) DebugBrowse(ctx context.Context, objectID string) error {
	result, numberReturned, totalMatches, updateID, err :=
		c.contentDir.BrowseCtx(
			ctx,
			objectID,
			"BrowseDirectChildren",
			"*",
			0,
			100,
			"",
		)

	if err != nil {
		return fmt.Errorf("browse %s: %w", objectID, err)
	}

	fmt.Printf("\n--- Sonos Browse: %s ---\n", objectID)
	fmt.Printf("Number returned: %d\n", numberReturned)
	fmt.Printf("Total matches:   %d\n", totalMatches)
	fmt.Printf("Update ID:       %d\n", updateID)
	fmt.Printf("Raw DIDL:\n%s\n", result)

	return nil
}
