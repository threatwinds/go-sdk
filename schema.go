package twsdk

type Alert struct {
	Id          *string  `json:"id,omitempty"`
	Timestamp   string   `json:"@timestamp"`
	LastUpdate  string   `json:"lastUpdate"`
	Name        string   `json:"name"`
	TenantId    string   `json:"tenantId"`
	TenantName  *string  `json:"tenantName,omitempty"`
	DataSource  string   `json:"dataSource"`
	DataType    string   `json:"dataType"`
	Category    string   `json:"category"`
	Technique   string   `json:"technique"`
	Description string   `yaml:"description"`
	References  []string `yaml:"references"`
	Impact      Impact   `json:"impact"`
	ImpactScore int      `json:"impactScore"`
	Severity    string   `json:"severity"`
	Adversary   *Side    `json:"adversary,omitempty"`
	Target      *Side    `json:"target,omitempty"`
	Events      []Event  `json:"events"`
}

type Notification struct {
	Id        *string `json:"id,omitempty"`
	Timestamp string  `json:"@timestamp"`
	Topic     string  `json:"topic"`
	Message   string  `json:"message"`
}

type Impact struct {
	Confidentiality int `json:"confidentiality"`
	Integrity       int `json:"integrity"`
	Availability    int `json:"availability"`
}

type Event struct {
	Id               *string                `json:"id,omitempty"`
	Timestamp        string                 `json:"@timestamp" example:"2022-09-28T18:39:28.000Z"`
	DeviceTime       string                 `json:"deviceTime" example:"2022-09-28T18:39:28.000Z"`
	DataType         string                 `json:"dataType" example:"linux"`
	DataSource       string                 `json:"dataSource" example:"192.168.1.245"`
	TenantId         string                 `json:"tenantId"`
	TenantName       *string                `json:"tenantName,omitempty"`
	Raw              *string                `json:"raw,omitempty"`
	Log              map[string]interface{} `json:"log,omitempty"`
	Remote           *Side                  `json:"remote,omitempty"`
	Local            *Side                  `json:"local,omitempty"`
	From             *Side                  `json:"from,omitempty"`
	To               *Side                  `json:"to,omitempty"`
	Protocol         *string                `json:"protocol,omitempty"`
	ConnectionStatus *string                `json:"connectionStatus,omitempty"`
	StatusCode       *int64                 `json:"statusCode,omitempty"`
}

type Geolocation struct {
	Country   *string  `json:"country,omitempty"`
	City      *string  `json:"city,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Asn       *int64   `json:"asn,omitempty"`
	Aso       *string  `json:"aso,omitempty"`
}

type Side struct {
	IP               *string       `json:"ip,omitempty"`
	IPs              []string      `json:"ips,omitempty"`
	Host             *string       `json:"host,omitempty"`
	Hosts            []string      `json:"hosts,omitempty"`
	User             *string       `json:"user,omitempty"`
	Users            []string      `json:"users,omitempty"`
	Group            *string       `json:"group,omitempty"`
	Groups           []string      `json:"groups,omitempty"`
	Port             *int64        `json:"port,omitempty"`
	Ports            []int64       `json:"ports,omitempty"`
	BytesSent        *float64      `json:"bytesSent,omitempty"`
	BytesReceived    *float64      `json:"bytesReceived,omitempty"`
	PackagesSent     *int64        `json:"packagesSent,omitempty"`
	PackagesReceived *int64        `json:"packagesReceived,omitempty"`
	Connections      *int64        `json:"connections,omitempty"`
	UsedCpuPercent   *int64        `json:"usedCpuPercent,omitempty"`
	UsedMemPercent   *int64        `json:"usedMemPercent,omitempty"`
	FreeCpuPercent   *int64        `json:"freeCpuPercent,omitempty"`
	FreeMemPercent   *int64        `json:"freeMemPercent,omitempty"`
	TotalCpuPercent  *int64        `json:"totalCpuPercent,omitempty"`
	TotalMemPercent  *int64        `json:"totalMemPercent,omitempty"`
	Domain           *string       `json:"domain,omitempty"`
	Domains          []string      `json:"domains,omitempty"`
	Fqdn             *string       `json:"fqdn,omitempty"`
	Fqdns            []string      `json:"fqdns,omitempty"`
	Mac              *string       `json:"mac,omitempty"`
	Macs             []string      `json:"macs,omitempty"`
	Process          *string       `json:"process,omitempty"`
	Processes        []string      `json:"processes,omitempty"`
	ASN              *int64        `json:"asn,omitempty"`
	ASO              *string       `json:"aso,omitempty"`
	Geolocations     []Geolocation `json:"geolocation,omitempty"`
	File             *string       `json:"file,omitempty"`
	Files            []string      `json:"files,omitempty"`
	Path             *string       `json:"path,omitempty"`
	Paths            []string      `json:"paths,omitempty"`
	MD5              *string       `json:"md5,omitempty"`
	MD5s             []string      `json:"md5s,omitempty"`
	SHA1             *string       `json:"sha1,omitempty"`
	SHA1s            []string      `json:"sha1s,omitempty"`
	SHA256           *string       `json:"sha256,omitempty"`
	SHA256s          []string      `json:"sha256s,omitempty"`
	URL              *string       `json:"url,omitempty"`
	URLs             []string      `json:"urls,omitempty"`
	Email            *string       `json:"email,omitempty"`
	Emails           []string      `json:"emails,omitempty"`
	Command          *string       `json:"command,omitempty"`
	Commands         []string      `json:"commands,omitempty"`
}
