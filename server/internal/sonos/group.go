package sonos

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	zoneGroupTopologyPath = "/ZoneGroupTopology/Control"

	zoneGroupTopologySOAPAction = `"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupState"`
)

type Group struct {
	ID            string        `json:"id"`
	CoordinatorID string        `json:"coordinatorId"`
	Members       []GroupMember `json:"members"`
}

type GroupMember struct {
	SpeakerID string `json:"speakerId"`
	Name      string `json:"name"`
}

type getZoneGroupStateResponse struct {
	Body struct {
		Response struct {
			ZoneGroupState string `xml:"ZoneGroupState"`
		} `xml:"GetZoneGroupStateResponse"`
	} `xml:"Body"`
}

type zoneGroupState struct {
	ZoneGroups struct {
		Groups []zoneGroup `xml:"ZoneGroup"`
	} `xml:"ZoneGroups"`
}

type zoneGroup struct {
	ID            string            `xml:"ID,attr"`
	CoordinatorID string            `xml:"Coordinator,attr"`
	Members       []zoneGroupMember `xml:"ZoneGroupMember"`
}

type zoneGroupMember struct {
	UUID     string `xml:"UUID,attr"`
	ZoneName string `xml:"ZoneName,attr"`
}

// DiscoverGroups retrieves the current Sonos group topology.
func DiscoverGroups(
	ctx context.Context,
	speakers []Speaker,
) ([]Group, error) {
	if len(speakers) == 0 {
		return nil, fmt.Errorf("no discovered speakers available")
	}

	// Any reachable Sonos speaker can provide the group topology.
	state, err := getZoneGroupState(ctx, speakers[0].IP)
	if err != nil {
		return nil, fmt.Errorf("discovering Sonos groups: %w", err)
	}

	var topology zoneGroupState
	if err := xml.Unmarshal([]byte(state), &topology); err != nil {
		return nil, fmt.Errorf("parsing Sonos group topology: %w", err)
	}

	// Map UPnP UUIDs to the speaker IDs returned by discovery.
	speakerIDs := make(map[string]string, len(speakers))
	for _, speaker := range speakers {
		speakerIDs[normaliseSpeakerID(speaker.ID)] = speaker.ID
	}

	groups := make([]Group, 0, len(topology.ZoneGroups.Groups))

	for _, zone := range topology.ZoneGroups.Groups {
		group := Group{
			ID:            zone.ID,
			CoordinatorID: resolveSpeakerID(zone.CoordinatorID, speakerIDs),
			Members:       make([]GroupMember, 0, len(zone.Members)),
		}

		for _, member := range zone.Members {
			group.Members = append(group.Members, GroupMember{
				SpeakerID: resolveSpeakerID(member.UUID, speakerIDs),
				Name:      member.ZoneName,
			})
		}

		groups = append(groups, group)
	}

	return groups, nil
}

func getZoneGroupState(
	ctx context.Context,
	speakerIP string,
) (string, error) {
	if net.ParseIP(speakerIP) == nil {
		return "", fmt.Errorf("invalid speaker IP address: %s", speakerIP)
	}

	endpoint := fmt.Sprintf(
		"http://%s%s",
		net.JoinHostPort(speakerIP, "1400"),
		zoneGroupTopologyPath,
	)

	soapBody := `<?xml version="1.0"?>` +
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body>` +
		`<u:GetZoneGroupState xmlns:u="urn:schemas-upnp-org:service:ZoneGroupTopology:1"></u:GetZoneGroupState>` +
		`</s:Body>` +
		`</s:Envelope>`

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(soapBody),
	)
	if err != nil {
		return "", fmt.Errorf("creating topology request: %w", err)
	}

	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPAction", zoneGroupTopologySOAPAction)

	resp, err := sonosHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting group topology: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("topology SOAP status code: %d", resp.StatusCode)
	}

	var soapResponse getZoneGroupStateResponse
	if err := xml.NewDecoder(resp.Body).Decode(&soapResponse); err != nil {
		return "", fmt.Errorf("decoding topology response: %w", err)
	}

	if soapResponse.Body.Response.ZoneGroupState == "" {
		return "", fmt.Errorf("empty group topology returned")
	}

	return soapResponse.Body.Response.ZoneGroupState, nil
}

func normaliseSpeakerID(id string) string {
	return strings.TrimPrefix(
		strings.ToLower(strings.TrimSpace(id)),
		"uuid:",
	)
}

func resolveSpeakerID(
	id string,
	speakerIDs map[string]string,
) string {
	if speakerID, ok := speakerIDs[normaliseSpeakerID(id)]; ok {
		return speakerID
	}

	return id
}
