package azure

type odata interface{}

type Location struct {
	DisplayName string
}

type Result struct {
	Next  string `json:"@odata.nextLink"`
	Value Events
}

type Events []Event

type Event struct {
	AllowNewTimeProposals         bool
	Attendees                     []odata              //": [{"@odata.type": "microsoft.graph.attendee"}],
	Body                          odata                //": {"@odata.type": "microsoft.graph.itemBody"},
	BodyPreview                   string               //": "string",
	Categories                    []string             //": ["string"],
	ChangeKey                     string               //": "string",
	CreatedDateTime               string               //": "String (timestamp)",
	End                           BullshitTime         //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
	HasAttachments                bool                 //": true,
	HideAttendees                 bool                 //": false,
	Id                            string               //": "string (identifier)",
	Importance                    string               //": "String",
	IsAllDay                      bool                 //": true,
	IsCancelled                   bool                 //": true,
	IsDraft                       bool                 //": false,
	IsOnlineMeeting               bool                 //": true,
	IsOrganizer                   bool                 //": true,
	IsReminderOn                  bool                 //": true,
	LastModifiedDateTime          string               //": "String (timestamp)",
	Location                      Location             //": {"@odata.type": "microsoft.graph.location"},
	Locations                     []odata              //": [{"@odata.type": "microsoft.graph.location"}],
	OnlineMeeting                 odata                //": {"@odata.type": "microsoft.graph.onlineMeetingInfo"},
	OnlineMeetingProvider         string               //": "string",
	OnlineMeetingUrl              string               //": "string",
	Organizer                     odata                //": {"@odata.type": "microsoft.graph.recipient"},
	OriginalEndTimeZone           string               //": "string",
	OriginalStart                 string               //": "String (timestamp)",
	OriginalStartTimeZone         string               //": "string",
	Recurrence                    *PatternedRecurrence //": {"@odata.type": "microsoft.graph.patternedRecurrence"},
	ReminderMinutesBeforeStart    int                  //": 1024,
	ResponseRequested             bool                 //": true,
	ResponseStatus                odata                //": {"@odata.type": "microsoft.graph.responseStatus"},
	Sensitivity                   string               //": "String",
	SeriesMasterId                string               //": "string",
	ShowAs                        string               //": "String",
	Start                         BullshitTime         //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
	Subject                       string               //": "string",
	Type                          string               //": "String",
	WebLink                       string               //": "string",
	Attachments                   []odata              //[ { "@odata.type": "microsoft.graph.attachment" } ],
	Calendar                      odata                //": { "@odata.type": "microsoft.graph.calendar" },
	Extensions                    []odata              //": [ { "@odata.type": "microsoft.graph.extension" } ],
	Instances                     []odata              //": [ { "@odata.type": "microsoft.graph.event" }],
	SingleValueExtendedProperties []odata              //": [ { "@odata.type": "microsoft.graph.singleValueLegacyExtendedProperty" }],
	MultiValueExtendedProperties  []odata              //": [ { "@odata.type": "microsoft.graph.multiValueLegacyExtendedProperty" }]
}

// start, end (string datetime)
// type, recurrence
// subject
// location.displayName
