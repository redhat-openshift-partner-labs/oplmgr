package internal

import (
	"net/url"
	"time"

	"github.com/google/uuid"
)

// RequestForm is used by pop to map your request_forms database table to your go code.
type RequestForm struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	Time                 string    `json:"time" db:"time"`
	Epoch                string    `json:"epoch" db:"epoch"`
	Clusterid            string    `json:"clusterid" db:"clusterid"`
	Generatedclustername string    `json:"generatedclustername" db:"generatedclustername"`
	Provider             string    `json:"provider" db:"provider"`
	Openshiftversion     string    `json:"openshiftversion" db:"openshiftversion"`
	Clustersize          string    `json:"clustersize" db:"clustersize"`
	Companyname          string    `json:"companyname" db:"companyname"`
	Connectpartner       string    `json:"connectpartner" db:"connectpartner"`
	Sponsor              string    `json:"sponsor" db:"sponsor"`
	Primaryname          string    `json:"primaryname" db:"primaryname"`
	Primaryemail         string    `json:"primaryemail" db:"primaryemail"`
	Primaryphone         string    `json:"primaryphone" db:"primaryphone"`
	Primaryconnect       string    `json:"primaryconnect" db:"primaryconnect"`
	Secondaryname        string    `json:"secondaryname" db:"secondaryname"`
	Secondaryemail       string    `json:"secondaryemail" db:"secondaryemail"`
	Secondaryphone       string    `json:"secondaryphone" db:"secondaryphone"`
	Secondaryconnect     string    `json:"secondaryconnect" db:"secondaryconnect"`
	Timezone             string    `json:"timezone" db:"timezone"`
	Projectname          string    `json:"projectname" db:"projectname"`
	Clustername          string    `json:"clustername" db:"clustername"`
	Reservation          string    `json:"reservation" db:"reservation"`
	Certproject          string    `json:"certproject" db:"certproject"`
	Intendedcertproject  string    `json:"intendedcertproject" db:"intendedcertproject"`
	Description          string    `json:"description" db:"description"`
	Notes                string    `json:"notes" db:"notes"`
	Startdate            string    `json:"startdate" db:"startdate"`
	Enddate              string    `json:"enddate" db:"enddate"`
	State                string    `json:"state" db:"state"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

type LabRequest struct {
	Timestamp                    string    `json:"time"`
	Epoch                        int       `json:"epoch" validate:"required"`
	ID                           uuid.UUID `json:"labid" validate:"omitempty"`
	LeaseTime                    int       `json:"leaseTime" validate:"omitempty"`
	PrimaryContactName           string    `json:"primaryContactName" validate:"required"`
	PrimaryContactEmail          string    `json:"primaryContactEmail" validate:"required,email"`
	PrimaryContactPhoneNumber    string    `json:"primaryContactPhoneNumber" validate:"omitempty"`
	PrimaryContactConnectUser    bool      `json:"isPrimaryContactConnectUser" validate:"omitempty"`
	SecondaryContactName         string    `json:"secondaryContactName" validate:"required"`
	SecondaryContactEmail        string    `json:"secondaryContactEmail" validate:"required,email"`
	SecondaryContactPhoneNumber  string    `json:"secondaryContactPhoneNumber" validate:"omitempty"`
	SecondaryContactConnectUser  bool      `json:"isSecondaryContactConnectUser" validate:"omitempty"`
	RedHatSponsor                string    `json:"redHatSponsor" validate:"required"`
	Availability                 string    `json:"availability" validate:"required"`
	CompanyName                  string    `json:"companyName" validate:"required"`
	CompanyConnectPartner        bool      `json:"isCompanyConnectPartner" validate:"omitempty"`
	CertificationProject         string    `json:"certificationProject" validate:"omitempty"`
	IntendedCertificationProject string    `json:"intendedCertificationProject" validate:"omitempty"`
	ProjectName                  string    `json:"projectName" validate:"omitempty"`
	PublicSSHKey                 string    `json:"publicsshkey" validate:"omitempty"`
	ClusterName                  string    `json:"clusterName" validate:"required"`
	ClusterSize                  int       `json:"clusterSize" validate:"omitempty"`
	OpenShiftVersion             string    `json:"openShiftVersion" validate:"required"`
	Description                  string    `json:"description" validate:"omitempty"`
	Notes                        string    `json:"notes" validate:"omitempty"`
}

// LabRequestBranch is the branch created when a LabRequest has been validated
// and approved. This branch is used when creating a PR for the LabRequest and
// is based on latest master
type LabRequestBranch struct {
	Base string `json:"base"`
	Lab  string `json:"labid"`
}

// LabRequestFile is the file generated for a pull request when a LabRequest
// has been validated and approved. This file is created prior to creating the
// pull request.
type LabRequestFile struct {
	FileName          string `json:"filename"`
	FileCommitMessage string `json:"filecommitmessage"`
	FileContent       string `json:"filecontent"`
}

type InstallConfig struct {
	BaseDomain        string
	WorkerReplicas    int
	MasterReplicas    int
	MasterSize        string
	WorkerSize        string
	ClusterName       string
	NetworkType       string
	ServiceNetwork    string
	Cloud             string
	RegionDesignation string
	Region            string
	PullSecret        string
	PublicSSHKey      string
}

type Alphabet struct {
	Decode [128]int8
	Encode [58]byte
}

type Cfg struct {
	Name             string `json:"name"`
	Host             string `json:"host"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	Expire           string `json:"expire"`
	OpenDiscussion   bool   `json:"open_discussion"`
	BurnAfterReading bool   `json:"burn_after_reading"`
	Formatter        string `json:"formatter"`
}

type PBClient struct {
	URL      url.URL
	Username string
	Password string
}

type CreatePasteRequest struct {
	V     int                    `json:"v"`
	AData []interface{}          `json:"adata"`
	Meta  CreatePasteRequestMeta `json:"meta"`
	CT    string                 `json:"ct"`
}

type CreatePasteRequestMeta struct {
	Expire string `json:"expire"`
}

type CreatePasteResponse struct {
	ID          string `json:"id"`
	Status      int    `json:"status"`
	Message     string `json:"message"`
	URL         string `json:"url"`
	DeleteToken string `json:"deletetoken"`
}

type PasteSpec struct {
	IV          string
	Salt        string
	Iterations  int
	KeySize     int
	TagSize     int
	Algorithm   string
	Mode        string
	Compression string
}

type PasteData struct {
	*PasteSpec
	Data             []byte
	Formatter        string
	OpenDiscussion   bool
	BurnAfterReading bool
}

type PasteContent struct {
	Paste string `json:"paste"`
}

type FormRequest struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body" validate:"required"`
}

type UtcTime struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Formatted string `json:"formatted"`
}
