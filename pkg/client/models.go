package client

type Email struct {
	Value   string `json:"value"`
	Type    string `json:"type"`
	Display string `json:"display"`
}

type PhoneNumber struct {
	Value   string `json:"value"`
	Type    string `json:"type"`
	Display string `json:"display"`
}

type Address struct {
	Type          string `json:"type"`
	Formatted     string `json:"formatted"`
	StreetAddress string `json:"street_address"`
	Locality      string `json:"locality"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
}

type Photo struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type Name struct {
	DisplayName string `json:"display_name"`
}

type User struct {
	ID                string        `json:"id"`
	CreatedAt         string        `json:"created_at"`
	UpdatedAt         string        `json:"updated_at"`
	Active            bool          `json:"active"`
	Username          string        `json:"username"`
	Name              Name          `json:"name"`
	Emails            []Email       `json:"emails"`
	PhoneNumbers      []PhoneNumber `json:"phone_numbers"`
	Addresses         []Address     `json:"addresses"`
	Photos            []Photo       `json:"photos"`
	PreferredLanguage string        `json:"preferred_language"`
	Locale            string        `json:"locale"`
	Timezone          string        `json:"timezone"`
	Number            string        `json:"number"`
}

type Meta struct {
	RedactedFields []RedactedField `json:"redacted_fields,omitempty"`
}

type RedactedField struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type UsersResponse struct {
	Meta     Meta   `json:"__meta"`
	Results  []User `json:"results"`
	NextLink string `json:"next_link,omitempty"`
}

type Team struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ParentID  string `json:"parent_id,omitempty"`
	Parent    *Team  `json:"parent,omitempty"`
	Name      string `json:"name"`
}

type TeamsResponse struct {
	Meta     Meta   `json:"__meta"`
	Results  []Team `json:"results"`
	NextLink string `json:"next_link,omitempty"`
}

// Worker-specific structs
type WorkerName struct {
	Formatted           string `json:"formatted,omitempty"`
	GivenName           string `json:"given_name,omitempty"`
	MiddleName          string `json:"middle_name,omitempty"`
	FamilyName          string `json:"family_name,omitempty"`
	PreferredGivenName  string `json:"preferred_given_name,omitempty"`
	PreferredFamilyName string `json:"preferred_family_name,omitempty"`
	DisplayName         string `json:"display_name,omitempty"`
}

type WorkerUser struct {
	ID                string        `json:"id"`
	CreatedAt         string        `json:"created_at"`
	UpdatedAt         string        `json:"updated_at"`
	Active            bool          `json:"active,omitempty"`
	Username          string        `json:"username,omitempty"`
	Name              WorkerName    `json:"name,omitempty"`
	Emails            []Email       `json:"emails,omitempty"`
	PhoneNumbers      []PhoneNumber `json:"phone_numbers,omitempty"`
	Addresses         []Address     `json:"addresses,omitempty"`
	Photos            []Photo       `json:"photos,omitempty"`
	PreferredLanguage string        `json:"preferred_language,omitempty"`
	Locale            string        `json:"locale,omitempty"`
	Timezone          string        `json:"timezone,omitempty"`
	Number            string        `json:"number,omitempty"`
}

type Country struct {
	Code string `json:"code,omitempty"`
}

type Company struct {
	ID                  string        `json:"id"`
	CreatedAt           string        `json:"created_at"`
	UpdatedAt           string        `json:"updated_at"`
	ParentLegalEntityID string        `json:"parent_legal_entity_id,omitempty"`
	ParentLegalEntity   *LegalEntity  `json:"parent_legal_entity,omitempty"`
	LegalEntitiesID     []string      `json:"legal_entities_id"`
	LegalEntities       []LegalEntity `json:"legal_entities,omitempty"`
	PhysicalAddress     *Address      `json:"physical_address,omitempty"`
	PrimaryEmail        string        `json:"primary_email,omitempty"`
	LegalName           string        `json:"legal_name,omitempty"`
	DoingBusinessAsName string        `json:"doing_business_as_name,omitempty"`
	Phone               string        `json:"phone,omitempty"`
	Name                string        `json:"name"`
	Country             string        `json:"country,omitempty"`
}

type LegalEntity struct {
	ID               string       `json:"id"`
	CreatedAt        string       `json:"created_at"`
	UpdatedAt        string       `json:"updated_at"`
	TaxIdentifier    string       `json:"tax_identifier,omitempty"`
	Country          *Country     `json:"country,omitempty"`
	LegalName        string       `json:"legal_name,omitempty"`
	EntityLevel      string       `json:"entity_level,omitempty"`
	RegistrationDate string       `json:"registration_date,omitempty"`
	MailingAddress   *Address     `json:"mailing_address,omitempty"`
	PhysicalAddress  *Address     `json:"physical_address,omitempty"`
	ParentID         string       `json:"parent_id,omitempty"`
	Parent           *LegalEntity `json:"parent,omitempty"`
	ManagementType   string       `json:"management_type,omitempty"`
	CompanyID        string       `json:"company_id,omitempty"`
	Company          *Company     `json:"company,omitempty"`
}

type Location struct {
	Type           string `json:"type"`
	WorkLocationID string `json:"work_location_id"`
}

type Currency struct {
	CurrencyType string  `json:"currency_type,omitempty"`
	Value        float64 `json:"value,omitempty"`
}

type Compensation struct {
	ID                       string    `json:"id"`
	CreatedAt                string    `json:"created_at"`
	UpdatedAt                string    `json:"updated_at"`
	WorkerID                 string    `json:"worker_id,omitempty"`
	Worker                   *Worker   `json:"worker,omitempty"`
	AnnualCompensation       *Currency `json:"annual_compensation,omitempty"`
	AnnualSalaryEquivalent   *Currency `json:"annual_salary_equivalent,omitempty"`
	HourlyWage               *Currency `json:"hourly_wage,omitempty"`
	MonthlyCompensation      *Currency `json:"monthly_compensation,omitempty"`
	OnTargetCommission       *Currency `json:"on_target_commission,omitempty"`
	RelocationReimbursement  *Currency `json:"relocation_reimbursement,omitempty"`
	SigningBonus             *Currency `json:"signing_bonus,omitempty"`
	TargetAnnualBonus        *Currency `json:"target_annual_bonus,omitempty"`
	WeeklyCompensation       *Currency `json:"weekly_compensation,omitempty"`
	TargetAnnualBonusPercent float64   `json:"target_annual_bonus_percent,omitempty"`
	BonusSchedule            string    `json:"bonus_schedule,omitempty"`
	PaymentType              string    `json:"payment_type,omitempty"`
	PaymentTerms             string    `json:"payment_terms,omitempty"`
	SalaryEffectiveDate      string    `json:"salary_effective_date,omitempty"`
	OvertimeExemption        string    `json:"overtime_exemption,omitempty"`
}

type EmploymentType struct {
	ID                     string `json:"id"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
	Label                  string `json:"label"`
	Name                   string `json:"name,omitempty"`
	Type                   string `json:"type,omitempty"`
	CompensationTimePeriod string `json:"compensation_time_period,omitempty"`
	AmountWorked           string `json:"amount_worked,omitempty"`
}

type Department struct {
	ID            string      `json:"id"`
	CreatedAt     string      `json:"created_at"`
	UpdatedAt     string      `json:"updated_at"`
	Name          string      `json:"name"`
	ParentID      string      `json:"parent_id,omitempty"`
	Parent        *Department `json:"parent,omitempty"`
	ReferenceCode string      `json:"reference_code,omitempty"`
}

type Track struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Name      string `json:"name"`
}

type Level struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Name        string `json:"name"`
	ParentID    string `json:"parent_id,omitempty"`
	Parent      *Level `json:"parent,omitempty"`
	GlobalLevel int    `json:"global_level,omitempty"`
	Description string `json:"description,omitempty"`
	Rank        int    `json:"rank,omitempty"`
	TrackID     string `json:"track_id,omitempty"`
	Track       *Track `json:"track,omitempty"`
}

type TerminationDetails struct {
	Type   string `json:"type,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type CountryFields struct {
	US *USFields `json:"us,omitempty"`
	CA *CAFields `json:"ca,omitempty"`
}

type USFields struct {
	SSN string `json:"ssn,omitempty"`
}

type CAFields struct {
	SIN string `json:"sin,omitempty"`
}

type BusinessPartnerGroup struct {
	ID string `json:"id,omitempty"`
}

type ClientGroup struct {
	ID string `json:"id,omitempty"`
}

type BusinessPartner struct {
	ID                     string                `json:"id"`
	CreatedAt              string                `json:"created_at"`
	UpdatedAt              string                `json:"updated_at"`
	BusinessPartnerGroupID string                `json:"business_partner_group_id"`
	BusinessPartnerGroup   *BusinessPartnerGroup `json:"business_partner_group,omitempty"`
	WorkerID               string                `json:"worker_id"`
	Worker                 *Worker               `json:"worker,omitempty"`
	ClientGroupID          string                `json:"client_group_id,omitempty"`
	ClientGroup            *ClientGroup          `json:"client_group,omitempty"`
	ClientGroupMemberCount int                   `json:"client_group_member_count,omitempty"`
}

type Worker struct {
	ID                 string              `json:"id"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
	UserID             string              `json:"user_id,omitempty"`
	User               *WorkerUser         `json:"user,omitempty"`
	IsManager          bool                `json:"is_manager,omitempty"`
	ManagerID          string              `json:"manager_id,omitempty"`
	Manager            *Worker             `json:"manager,omitempty"`
	LegalEntityID      string              `json:"legal_entity_id,omitempty"`
	LegalEntity        *LegalEntity        `json:"legal_entity,omitempty"`
	Country            string              `json:"country,omitempty"`
	StartDate          string              `json:"start_date,omitempty"`
	EndDate            string              `json:"end_date,omitempty"`
	Number             int                 `json:"number,omitempty"`
	WorkEmail          string              `json:"work_email,omitempty"`
	PersonalEmail      string              `json:"personal_email,omitempty"`
	Status             string              `json:"status,omitempty"`
	Location           *Location           `json:"location,omitempty"`
	EmploymentTypeID   string              `json:"employment_type_id,omitempty"`
	EmploymentType     *EmploymentType     `json:"employment_type,omitempty"`
	Gender             string              `json:"gender,omitempty"`
	DateOfBirth        string              `json:"date_of_birth,omitempty"`
	Race               string              `json:"race,omitempty"`
	Ethnicity          string              `json:"ethnicity,omitempty"`
	Citizenship        string              `json:"citizenship,omitempty"`
	CompensationID     string              `json:"compensation_id,omitempty"`
	Compensation       *Compensation       `json:"compensation,omitempty"`
	DepartmentID       string              `json:"department_id,omitempty"`
	Department         *Department         `json:"department,omitempty"`
	TeamsID            []string            `json:"teams_id,omitempty"`
	Teams              []Team              `json:"teams,omitempty"`
	Title              string              `json:"title,omitempty"`
	TitleEffectiveDate string              `json:"title_effective_date,omitempty"`
	LevelID            string              `json:"level_id,omitempty"`
	Level              *Level              `json:"level,omitempty"`
	TerminationDetails *TerminationDetails `json:"termination_details,omitempty"`
	CustomFields       []interface{}       `json:"custom_fields,omitempty"`
	CountryFields      *CountryFields      `json:"country_fields,omitempty"`
	BusinessPartnersID []string            `json:"business_partners_id,omitempty"`
	BusinessPartners   []BusinessPartner   `json:"business_partners,omitempty"`
}

type WorkersResponse struct {
	Meta     Meta     `json:"__meta"`
	Results  []Worker `json:"results"`
	NextLink string   `json:"next_link,omitempty"`
}
