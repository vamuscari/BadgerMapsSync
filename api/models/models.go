package models

import (
	"encoding/json"

	"github.com/guregu/null/v6"
)

// AccountUpload contains form fields used for creating/updating an account.
type AccountUpload struct {
	Fields map[string]string `json:"fields"`
}

// CheckinUpload contains form fields used for creating a checkin.
type CheckinUpload struct {
	Customer int               `json:"customer"`
	Type     string            `json:"type"`
	Fields   map[string]string `json:"fields"`
}

// CustomCheckinUpload contains form fields used for creating a custom checkin.
type CustomCheckinUpload struct {
	Customer    int                       `json:"customer"`
	Type        string                    `json:"type"`
	Fields      map[string]string         `json:"fields"`
	ExtraFields *CustomCheckinExtraFields `json:"extra_fields,omitempty"`
}

// CustomCheckinExtraFields contains typed custom checkin extra_fields values.
// All fields are optional.
type CustomCheckinExtraFields struct {
	LogType      string `json:"Log Type,omitempty"`
	MeetingNotes string `json:"Meeting Notes,omitempty"`
}

// LocationUpload contains form fields used for updating a location.
type LocationUpload struct {
	Fields map[string]string `json:"fields"`
}

// Account represents a BadgerMaps account (customer)
type Account struct {
	AccountId            null.Int     `json:"id"`
	FirstName            *null.String `json:"first_name"`
	LastName             null.String  `json:"last_name"`
	FullName             null.String  `json:"full_name"`
	PhoneNumber          null.String  `json:"phone_number"`
	Email                null.String  `json:"email"`
	CustomerId           *null.String `json:"customer_id"`
	Notes                *null.String `json:"notes"`
	OriginalAddress      null.String  `json:"original_address"`
	CrmId                *null.String `json:"crm_id"`
	AccountOwner         *null.String `json:"account_owner"`
	DaysSinceLastCheckin null.Int     `json:"days_since_last_checkin"`
	LastCheckinDate      *null.String `json:"last_checkin_date"`
	LastModifiedDate     *null.String `json:"last_modified_date"`
	FollowUpDate         *null.String `json:"follow_up_date"`
	Locations            []Location   `json:"locations"`
	CustomNumeric        *null.Float  `json:"custom_numeric"`
	CustomText           *null.String `json:"custom_text"`
	CustomNumeric2       *null.Float  `json:"custom_numeric2"`
	CustomText2          *null.String `json:"custom_text2"`
	CustomNumeric3       *null.Float  `json:"custom_numeric3"`
	CustomText3          *null.String `json:"custom_text3"`
	CustomNumeric4       *null.Float  `json:"custom_numeric4"`
	CustomText4          *null.String `json:"custom_text4"`
	CustomNumeric5       *null.Float  `json:"custom_numeric5"`
	CustomText5          *null.String `json:"custom_text5"`
	CustomNumeric6       *null.Float  `json:"custom_numeric6"`
	CustomText6          *null.String `json:"custom_text6"`
	CustomNumeric7       *null.Float  `json:"custom_numeric7"`
	CustomText7          *null.String `json:"custom_text7"`
	CustomNumeric8       *null.Float  `json:"custom_numeric8"`
	CustomText8          *null.String `json:"custom_text8"`
	CustomNumeric9       *null.Float  `json:"custom_numeric9"`
	CustomText9          *null.String `json:"custom_text9"`
	CustomNumeric10      *null.Float  `json:"custom_numeric10"`
	CustomText10         *null.String `json:"custom_text10"`
	CustomNumeric11      *null.Float  `json:"custom_numeric11"`
	CustomText11         *null.String `json:"custom_text11"`
	CustomNumeric12      *null.Float  `json:"custom_numeric12"`
	CustomText12         *null.String `json:"custom_text12"`
	CustomNumeric13      *null.Float  `json:"custom_numeric13"`
	CustomText13         *null.String `json:"custom_text13"`
	CustomNumeric14      *null.Float  `json:"custom_numeric14"`
	CustomText14         *null.String `json:"custom_text14"`
	CustomNumeric15      *null.Float  `json:"custom_numeric15"`
	CustomText15         *null.String `json:"custom_text15"`
	CustomNumeric16      *null.Float  `json:"custom_numeric16"`
	CustomText16         *null.String `json:"custom_text16"`
	CustomNumeric17      *null.Float  `json:"custom_numeric17"`
	CustomText17         *null.String `json:"custom_text17"`
	CustomNumeric18      *null.Float  `json:"custom_numeric18"`
	CustomText18         *null.String `json:"custom_text18"`
	CustomNumeric19      *null.Float  `json:"custom_numeric19"`
	CustomText19         *null.String `json:"custom_text19"`
	CustomNumeric20      *null.Float  `json:"custom_numeric20"`
	CustomText20         *null.String `json:"custom_text20"`
	CustomNumeric21      *null.Float  `json:"custom_numeric21"`
	CustomText21         *null.String `json:"custom_text21"`
	CustomNumeric22      *null.Float  `json:"custom_numeric22"`
	CustomText22         *null.String `json:"custom_text22"`
	CustomNumeric23      *null.Float  `json:"custom_numeric23"`
	CustomText23         *null.String `json:"custom_text23"`
	CustomNumeric24      *null.Float  `json:"custom_numeric24"`
	CustomText24         *null.String `json:"custom_text24"`
	CustomNumeric25      *null.Float  `json:"custom_numeric25"`
	CustomText25         *null.String `json:"custom_text25"`
	CustomNumeric26      *null.Float  `json:"custom_numeric26"`
	CustomText26         *null.String `json:"custom_text26"`
	CustomNumeric27      *null.Float  `json:"custom_numeric27"`
	CustomText27         *null.String `json:"custom_text27"`
	CustomNumeric28      *null.Float  `json:"custom_numeric28"`
	CustomText28         *null.String `json:"custom_text28"`
	CustomNumeric29      *null.Float  `json:"custom_numeric29"`
	CustomText29         *null.String `json:"custom_text29"`
	CustomNumeric30      *null.Float  `json:"custom_numeric30"`
	CustomText30         *null.String `json:"custom_text30"`
	CreatedAt            null.String  `json:"created_at"`
	UpdatedAt            null.String  `json:"updated_at"`
}

// Location represents a BadgerMaps location
type Location struct {
	LocationId    null.Int     `json:"id"`
	City          null.String  `json:"city"`
	Name          *null.String `json:"name"`
	Zipcode       null.String  `json:"zipcode"`
	Long          null.Float   `json:"long"`
	State         null.String  `json:"state"`
	Lat           null.Float   `json:"lat"`
	AddressLine1  null.String  `json:"address_line_1"`
	Location      null.String  `json:"location"`
	IsApproximate null.Bool    `json:"is_approximate"`
}

// Route represents a BadgerMaps route
type Route struct {
	RouteId            null.Int    `json:"id"`
	Name               null.String `json:"name"`
	RouteDate          null.String `json:"route_date"`
	Duration           *null.Int   `json:"duration"`
	Waypoints          []Waypoint  `json:"waypoints"`
	StartAddress       null.String `json:"start_address"`
	DestinationAddress null.String `json:"destination_address"`
	StartTime          null.String `json:"start_time"`
}

// Waypoint represents a route waypoint
type Waypoint struct {
	WaypointID      null.Int     `json:"id"`
	Name            null.String  `json:"name"`
	Address         null.String  `json:"address"`
	Suite           *null.String `json:"suite"`
	City            *null.String `json:"city"`
	State           *null.String `json:"state"`
	Zipcode         *null.String `json:"zipcode"`
	Location        null.String  `json:"location"`
	Lat             null.Float   `json:"lat"`
	Long            null.Float   `json:"long"`
	LayoverMinutes  null.Int     `json:"layover_minutes"`
	Position        null.Int     `json:"position"`
	CompleteAddress *null.String `json:"complete_address"`
	LocationID      null.Int     `json:"location_id"`
	CustomerID      null.Int     `json:"customer_id"`
	ApptTime        *null.String `json:"appt_time"`
	Type            null.Int     `json:"type"`
	PlaceID         *null.String `json:"place_id"`
}

// Checkin represents a BadgerMaps checkin (appointment)
type Checkin struct {
	CheckinId    null.Int        `json:"id"`
	CrmId        *null.String    `json:"crm_id"`
	AccountId    null.Int        `json:"customer"`
	LogDatetime  null.String     `json:"log_datetime"`
	Type         null.String     `json:"type"`
	Comments     null.String     `json:"comments"`
	ExtraFields  json.RawMessage `json:"extra_fields"`
	EndpointType null.String     `json:"endpoint_type"`
	CreatedBy    null.String     `json:"created_by"`
}

// UserProfile represents a BadgerMaps user profile
type UserProfile struct {
	ProfileId                 null.Int      `json:"id"`
	Email                     null.String   `json:"email"`
	FirstName                 null.String   `json:"first_name"`
	LastName                  null.String   `json:"last_name"`
	IsManager                 null.Bool     `json:"is_manager"`
	IsHideReferralIOSBanner   null.Bool     `json:"is_hide_referral_ios_banner"`
	MarkerIcon                null.String   `json:"marker_icon"`
	Manager                   *null.String  `json:"manager"`
	CRMEditableFieldsList     []null.String `json:"crm_editable_fields_list"`
	CRMBaseURL                null.String   `json:"crm_base_url"`
	CRMType                   null.String   `json:"crm_type"`
	ReferralURL               null.String   `json:"referral_url"`
	MapStartZoom              null.Int      `json:"map_start_zoom"`
	MapStart                  null.String   `json:"map_start"`
	IsUserCanEdit             null.Bool     `json:"is_user_can_edit"`
	IsUserCanDeleteCheckins   null.Bool     `json:"is_user_can_delete_checkins"`
	IsUserCanAddNewTextValues null.Bool     `json:"is_user_can_add_new_text_values"`
	HasData                   null.Bool     `json:"has_data"`
	DefaultApptLength         null.Int      `json:"default_appt_length"`
	Completed                 null.Bool     `json:"completed"`
	TrialDaysLeft             null.Int      `json:"trial_days_left"`
	ApptlogFields             []DataField   `json:"apptlog_fields"`
	AcctlogFields             []DataField   `json:"acctlog_fields"`
	Datafields                []DataField   `json:"datafields"`
	Company                   Company       `json:"company"`
}

// Company represents a BadgerMaps company
type Company struct {
	Id        null.Int    `json:"id"`
	ShortName null.String `json:"short_name"`
	Name      null.String `json:"name"`
}

// DataField represents a custom data field
type DataField struct {
	Name                      null.String  `json:"name"`
	Filterable                null.Bool    `json:"filterable"`
	Label                     null.String  `json:"label"`
	Values                    []FieldValue `json:"values,omitempty"`
	Position                  null.Int     `json:"position"`
	Type                      null.String  `json:"type"`
	HasData                   null.Bool    `json:"has_data"`
	IsUserCanAddNewTextValues null.Bool    `json:"is_user_can_add_new_text_values"`
	RawMin                    *null.Float  `json:"rawmin,omitempty"`
	Min                       *null.Float  `json:"min,omitempty"`
	Max                       *null.Float  `json:"max,omitempty"`
	RawMax                    *null.Float  `json:"rawmax,omitempty"`
	AccountField              null.String  `json:"account_field"`
}

// FieldValue represents a field value option
type FieldValue struct {
	Text  null.String `json:"text"`
	Value interface{} `json:"value"`
}
