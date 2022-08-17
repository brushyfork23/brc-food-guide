package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const campsFilename = "brc_api_2022/camps.json"
const artFilename = "brc_api_2022/art.json"
const eventsFilename = "brc_api_2022/events.json"
const outputFileName = "out/food-guide.txt"

const timestampLayout = "2006-01-02T15:04:05-07:00"

type CampLocation struct {
    String string `json:"string"`
    Frontage string `json:"frontage"`
    Intersection string `json:"intersection"`
    IntersectionType string `json:"intersection_type"`
}

type Camp struct {
    Uid string `json:"uid"`
    Name string `json:"name"`
    Description string `json:"description"`
    Location CampLocation `json:"location"`
    LocationString string `json:"location_string"`
}

type ArtLocation struct {
    Hour int16 `json:"hour"`
    Minute int16 `json:"minute"`
    Distance int32 `json:"distance"`
}

type Art struct {
    Uid string `json:"uid"`
    Name string `json:"name"`
    Location ArtLocation `json:"location"`
}

type EventType struct {
    ID int64 `json:"id"`
    Label string `json:"label"`
    Abbr string `json:"abbr"`
}

type Occurrence struct {
    Start string `json:"start_time"`
    End string `json:"end_time"`
}

type Event struct {
    Id int64 `json:"event_id"`
    Title string `json:"title"`
    Uid string `json:"uid"`
    Description string `json:"description"`
    EventType EventType `json:"event_type"`
    Year int32 `json:"year"`
    PrintDescription string `json:"print_description"`
    HostedByCamp string `json:"hosted_by_camp"`
    LocatedAtArt string `json:"located_at_art"`
    OtherLocation string `json:"other_location"`
    OccurrenceSet []Occurrence `json:"occurrence_set"`
}

type FormattedEvent struct {
    ID int64
    Times string
    Duration time.Duration
    EventName string
    EventDescription string
    Address string
    LocationName string
}

func main() {
    fmt.Println("Parsing Camps")

    campsFile, err := os.Open(campsFilename)
    if err != nil {
        panic(err)
    }
    defer campsFile.Close()

	decoder := json.NewDecoder(campsFile)

    var camps = make(map[string]Camp)

    // Read the array open bracket
	if _, err = decoder.Token(); err != nil {
		panic(fmt.Sprintf("Failed to decode initial array open bracket: %+v", err))
	}
	
	for decoder.More() {
        var camp Camp
		if err = decoder.Decode(&camp); err != nil {
			panic(fmt.Sprintf("Failed to decode line: %+v", err))
		}

		camps[camp.Uid] = camp
	}

    fmt.Println("Parsing Art")

    artFile, err := os.Open(artFilename)
    if err != nil {
        panic(err)
    }
    defer artFile.Close()

	decoder = json.NewDecoder(artFile)

    var arts = make(map[string]Art)

    // Read the array open bracket
	if _, err = decoder.Token(); err != nil {
		panic(fmt.Sprintf("Failed to decode initial array open bracket: %+v", err))
	}
	
	for decoder.More() {
        var art Art
		if err = decoder.Decode(&art); err != nil {
			panic(fmt.Sprintf("Failed to decode line: %+v", err))
		}

		arts[art.Uid] = art
	}

    fmt.Println("Parsing Events")

    eventsFile, err := os.Open(eventsFilename)
    if err != nil {
        panic(err)
    }
    defer eventsFile.Close()

	decoder = json.NewDecoder(eventsFile)

    // map[day][start_time][]events_sorted_by_length
    formattedEvents := make(map[string]map[string][]FormattedEvent)

    // Read the array open bracket
	if _, err = decoder.Token(); err != nil {
		panic(fmt.Sprintf("Failed to decode initial array open bracket: %+v", err))
	}
	
    entries := 0

	for decoder.More() {
        var event Event
		if err = decoder.Decode(&event); err != nil {
			panic(fmt.Sprintf("Failed to decode line: %+v", err))
		}

        if event.EventType.ID != 5 {
            continue
        }

        for _, occurrence := range event.OccurrenceSet {
            entries++

            startTime, err := time.Parse(timestampLayout, occurrence.Start)
            if err != nil {
                panic(fmt.Sprintf("Failed to parse time \"%s\": %v+",occurrence.Start,err))
            }
            endTime, err := time.Parse(timestampLayout, occurrence.End)
            if err != nil {
                panic(fmt.Sprintf("Failed to parse time \"%s\": %v+",occurrence.End,err))
            }
            dayKey := startTime.Format("01/02")
            timeKey := startTime.Format("1504")
            duration := endTime.Sub(startTime)
            camp := Camp{}
            if event.HostedByCamp != "" {
                _, found := camps[event.HostedByCamp]
                if !found {
                    fmt.Printf("Could not find camp for event %d\n", event.Id)
                } else {
                    camp = camps[event.HostedByCamp]
                }
            }
            var art Art
            if event.LocatedAtArt != "" {
                _, found := arts[event.LocatedAtArt]
                if !found {
                    fmt.Printf("Could not find art for event %d\n", event.Id)
                } else {
                    art = arts[event.LocatedAtArt]
                }
            }

            // "08/28 1800 - 08/28 1815",
            formatMinimumMinutes := func(t time.Time) string {
                if t.Minute() == 0 {
                    return t.Format("3pm")
                } else {
                    return t.Format("3:04pm")
                }
            }

            optionalSecondDate := ""
            if (startTime.Day() != endTime.Day()) {
                optionalSecondDate = endTime.Format("1/2 ")
            }

            timeString := fmt.Sprintf("%s %s - %s%s", startTime.Format("1/2"), formatMinimumMinutes(startTime), optionalSecondDate, formatMinimumMinutes(endTime))
            
            address := ""
            locationName := ""
            if camp.LocationString != "" {
                address = camp.LocationString
                locationName = camp.Name
            } else if (art != Art{}) {
                address = fmt.Sprintf("%d:%02d %d'", art.Location.Hour, art.Location.Minute, art.Location.Distance)
                locationName = art.Name
            } else {
                locationName = event.OtherLocation
            }

            if address == "" && locationName == "" {
                fmt.Printf("No address found for event %d\n", event.Id)
            }

            var formattedEvent = FormattedEvent{
                ID: event.Id,
                Times: timeString,
                Duration: duration,
                EventName: event.Title,
                EventDescription: event.Description,
                Address: address,
                LocationName: locationName,
            }

            day, exists := formattedEvents[dayKey]
            if !exists {
                formattedEvents[dayKey] = make(map[string][]FormattedEvent)
            }
            formattedEvents[dayKey][timeKey] = append(day[timeKey], formattedEvent)
        }
	}

    fmt.Println("Finished Parsing")

    fmt.Println("Sorting")

    for dayKey := range formattedEvents {
        for timeKey := range formattedEvents[dayKey] {
            sort.Slice(formattedEvents[dayKey][timeKey], func(i, j int) bool {
                return formattedEvents[dayKey][timeKey][i].Duration < formattedEvents[dayKey][timeKey][j].Duration
            })
        }
    }

    fmt.Println("Writing sorted, formatted events to file")

    outFile, err := os.Create(outputFileName)
	if err != nil {
		panic(fmt.Sprintf("Failed to open file for writing: %+v", err))
	}
	defer outFile.Close()

    dayKeys := make([]string, 0, len(formattedEvents))
    for k := range formattedEvents{
        dayKeys = append(dayKeys, k)
    }
    sort.Strings(dayKeys)

    for _, dayKey := range dayKeys {

        timeKeys := make([]string, 0, len(formattedEvents[dayKey]))
        for k := range formattedEvents[dayKey]{
            timeKeys = append(timeKeys, k)
        }
        sort.Strings(timeKeys)

        for _, timeKey := range timeKeys {
            for _, entry := range formattedEvents[dayKey][timeKey] {
                outFile.WriteString(
                    fmt.Sprintf("%s\t%s\t%s\n\t%s\n",
                        entry.Times,
                        entry.Address,
                        entry.LocationName,
                        word_wrap(
                            fmt.Sprintf("%s:  %s", entry.EventName, entry.EventDescription),
                            120,
                            "\t",
                        ),
                    ),
                )
            }
        }
    }

    fmt.Printf("Complete!  Wrote %d entries.\n", entries)
}

// Wraps text at the specified column lineWidth on word breaks
func word_wrap(text string, lineWidth int, linePrefix string) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text
	}
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + linePrefix + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped

}