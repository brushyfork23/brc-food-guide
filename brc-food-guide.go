package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/jung-kurt/gofpdf"
)

const campsFilename = "brc_api_2022/camps.json"
const artFilename = "brc_api_2022/art.json"
const eventsFilename = "brc_api_2022/events.json"
const outputFileName = "out/food-guide.pdf"

const lineHeight = 4

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
    Day string
    ShortTimes string
    LongTimes string
    StartTime string
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

    longestTime := ""
    longestAddress := ""
    longestLocationName := ""
    longestEventName := ""
    longestDescription := ""

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
	
    events := 0
    entries := 0

	for decoder.More() {
        var event Event
		if err = decoder.Decode(&event); err != nil {
			panic(fmt.Sprintf("Failed to decode line: %+v", err))
		}

        if event.EventType.ID != 5 {
            continue
        }

        events++

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
                optionalSecondDate = endTime.Format("Mon ")
            }

            dayString := startTime.Format("Mon 1/2")
            shortTimeString := fmt.Sprintf("%s - %s%s", formatMinimumMinutes(startTime), optionalSecondDate, formatMinimumMinutes(endTime)) 
            longTimeString := fmt.Sprintf("%s %s - %s%s", startTime.Format("Mon"), formatMinimumMinutes(startTime), optionalSecondDate, formatMinimumMinutes(endTime))
            
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
                Day: dayString,
                ShortTimes: shortTimeString,
                LongTimes: longTimeString,
                StartTime: fmt.Sprintf("%s %s", startTime.Format("Mon 1/2 "), formatMinimumMinutes(startTime)),
                Duration: duration,
                EventName: event.Title,
                EventDescription: event.Description,
                Address: address,
                LocationName: locationName,
            }

            // track longest entries to allow PDF formatting
            if utf8.RuneCountInString(longestTime) < utf8.RuneCountInString(formattedEvent.LongTimes) {
                longestTime = formattedEvent.LongTimes
            }
            if utf8.RuneCountInString(longestAddress) < utf8.RuneCountInString(formattedEvent.Address) {
                longestAddress = formattedEvent.Address
            }
            if utf8.RuneCountInString(longestLocationName) < utf8.RuneCountInString(formattedEvent.LocationName) {
                longestLocationName = formattedEvent.LocationName
            }
            if utf8.RuneCountInString(longestEventName) < utf8.RuneCountInString(formattedEvent.EventName) {
                longestEventName = formattedEvent.EventName
            }
            if utf8.RuneCountInString(longestDescription) < utf8.RuneCountInString(formattedEvent.EventDescription) {
                longestDescription = formattedEvent.EventDescription
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

    footerStartTime := ""

    var footerStartTimePointer *string

    footerStartTimePointer = &footerStartTime

    // Setup Document
    pdf := gofpdf.New("P", "mm", "Letter", "")
    make_title_page(pdf)
    pdf.AddPage()
    pdf.SetFont("Arial", "", 8)
    pdf.SetLeftMargin(14)
    tr := pdf.UnicodeTranslatorFromDescriptor("")

    pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 10, *footerStartTimePointer,
			"", 0, "C", false, 0, "")
	})

    dayKeys := make([]string, 0, len(formattedEvents))
    for k := range formattedEvents{
        dayKeys = append(dayKeys, k)
    }
    sort.Strings(dayKeys)

    lastPage := -1
    lastDay := ""

    for _, dayKey := range dayKeys {

        timeKeys := make([]string, 0, len(formattedEvents[dayKey]))
        for k := range formattedEvents[dayKey]{
            timeKeys = append(timeKeys, k)
        }
        sort.Strings(timeKeys)

        for _, timeKey := range timeKeys {
            for _, entry := range formattedEvents[dayKey][timeKey] {
                if lastPage < pdf.PageNo() {
                    lastPage = pdf.PageNo()
                    *footerStartTimePointer = entry.StartTime
                }
                
                pdf.SetX(12)
                pdf.SetFontSize(13)
                pdf.Write(lineHeight, tr("\u2022"))
                pdf.SetFontSize(8)
                pdf.SetX(14)
                pdf.SetFontStyle("B")
                pdf.Write(lineHeight, tr(entry.EventName))
                pdf.SetFontStyle("")
                pdf.SetX(pdf.GetX()+3)
                displayTime := entry.ShortTimes
                if lastDay != entry.Day {
                    displayTime = entry.LongTimes
                    lastDay = entry.Day
                }
                pdf.Write(lineHeight, displayTime)
                pdf.SetX(pdf.GetX()+4)
                if entry.Address != "" {
                    pdf.Write(lineHeight, fmt.Sprintf("(%s)",tr(entry.Address)))
                    pdf.SetX(pdf.GetX()+4)
                }
                pdf.SetFontStyle("I")
                pdf.Write(lineHeight, tr(entry.LocationName))
                pdf.SetFontStyle("")
                pdf.Write(lineHeight, "\n")
                pdf.Write(lineHeight, tr(entry.EventDescription))
                pdf.Write(lineHeight+2, "\n")
            }
        }
    }

    fmt.Printf("longestTime: %s\n", longestTime)
    fmt.Printf("longestAddress: %s\n", longestAddress)
    fmt.Printf("longestLocationName: %s\n", longestLocationName)
    fmt.Printf("longestEventName: %s\n", longestEventName)
    fmt.Printf("longestDescription: %s\n", longestDescription)

    err = pdf.OutputFileAndClose(outputFileName)
    if err != nil {
        panic(fmt.Sprintf("Failed to write PDF: %+v", err))
    }

    fmt.Printf("Complete!  Wrote %d occurrences of %d events.\n", events, entries)
}

func make_title_page(pdf *gofpdf.Fpdf) {
    pdf.AddPage()
    pdf.SetY(35)
    pdf.SetFont("Arial", "B", 24)
    pdf.WriteAligned(0, 20, "Hungry?               Bored?", "C")

    pdf.SetFont("Arial", "", 12)
    pdf.SetY(65)
    pdf.WriteAligned(0, 14, "Get a fork!                                                            ", "C")
    pdf.SetY(73)
    pdf.WriteAligned(0, 14, "Get a friend!                    ", "C")
    pdf.SetY(81)
    pdf.WriteAligned(0, 14, "                    Get a bib!", "C")
    pdf.SetY(89)
    pdf.WriteAligned(0, 14, "                                                        Get to the", "C")

    pdf.SetY(117)
    pdf.SetFont("Arial", "B", 38)
    pdf.WriteAligned(0, 38, "BRC 2022", "C")

    pdf.SetY(137)
    pdf.SetFont("Arial", "B", 40)
    pdf.WriteAligned(0, 40, "FOOD EVENTS", "C")

    pdf.Image("tautology-logo-small.png", 106, 230, 0, 0, false, "", 0, "")
}