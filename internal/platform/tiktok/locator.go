package tiktok

type PageLocators struct {
	Iframe          string
	Container       string
	UploadButton    []string
	FileInput       []string
	PostButton      []string
	Editor          []string
	CoverContainer  []string
	UploadCover     []string
	ConfirmCover    []string
	ScheduleButton  []string
	AllowButton     []string
	DatePicker      string
	CalendarWrapper string
	MonthTitle      string
	ArrowButton     string
	ValidDay        string
	TimePicker      string
	HourPicker      string
	MinutePicker    string
	SuccessModal    string
	VideoList       string
	VideoLink       string
	NavMoreMenu     string
	LanguageSelect  string
	EnglishOption   string
}

type LanguageLocators struct {
	English PageLocators
}

var Locators = LanguageLocators{
	English: PageLocators{
		Iframe: `iframe[data-tt='Upload_index_iframe']`,
		Container: `div.upload-container`,
		UploadButton: []string{
			`button:has-text('Select video'):visible`,
			`button[aria-label='Select file']`,
		},
		FileInput: []string{
			`input[type="file"]`,
			`input[accept*="video"]`,
		},
		PostButton: []string{
			`div.btn-post > button`,
			`div.button-group > button >> text=Post`,
		},
		Editor: []string{
			`div.public-DraftEditor-content`,
			`div[contenteditable="true"]`,
		},
		CoverContainer: []string{
			`.cover-container`,
			`.thumbnail-container`,
		},
		UploadCover: []string{
			`text=Upload cover`,
			`button:has-text("Upload cover")`,
		},
		ConfirmCover: []string{
			`text=Confirm`,
			`button:has-text("Confirm")`,
		},
		ScheduleButton: []string{
			`[aria-label="Schedule"]`,
			`button:has-text("Schedule")`,
		},
		AllowButton: []string{
			`div.TUXButton-content >> text=Allow`,
			`button:has-text("Allow")`,
		},
		DatePicker: `div.scheduled-picker div.TUXInputBox`,
		CalendarWrapper: `div.calendar-wrapper`,
		MonthTitle: `span.month-title`,
		ArrowButton: `span.arrow`,
		ValidDay: `span.day.valid`,
		TimePicker: `div.scheduled-picker div.TUXInputBox`,
		HourPicker: `span.tiktok-timepicker-left`,
		MinutePicker: `span.tiktok-timepicker-right`,
		SuccessModal: `#\\:r9\\:`,
		VideoList: `div[data-tt="components_PostTable_Container"]`,
		VideoLink: `div[data-tt="components_PostInfoCell_Container"] a`,
		NavMoreMenu: `[data-e2e="nav-more-menu"]`,
		LanguageSelect: `[data-e2e="language-select"]`,
		EnglishOption: `#creator-tools-selection-menu-header >> text=English (US)`,
	},
}

func GetLocators() *PageLocators {
	return &Locators.English
}

type UploadStatusLocators struct {
	ProgressSelector   string
	SuccessSelector    string
	ErrorSelector      string
	UploadingIndicator string
}

var UploadStatus = UploadStatusLocators{
	ProgressSelector:   `[class*="progress"], [class*="uploading"]`,
	SuccessSelector:    `text=/上传成功|上传完成|Upload complete|Processing/`,
	ErrorSelector:      `text=/上传失败|上传出错|Upload failed|Error/`,
	UploadingIndicator: `text=上传中`,
}
