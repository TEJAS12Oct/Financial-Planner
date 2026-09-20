package pdf

import (
	"FinancialPlannerGo/app/utils"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

const (
	hkGroteskFontFamily = "HKGrotesk"

	pageWidth    = 210.0
	pageHeight   = 297.0
	leftMargin   = 18.0
	rightMargin  = 18.0
	contentWidth = pageWidth - leftMargin - rightMargin

	// Extracted from the original AI Financial Plan Report.
	brownR  = 151
	brownG  = 58
	brownB  = 18
	orangeR = 189
	orangeG = 76
	orangeB = 35
)

//go:embed fonts/HKGrotesk-Regular.ttf fonts/HKGrotesk-Bold.ttf
var hkGroteskFonts embed.FS

type Generator struct {
	LogoPath    string
	TemplateDir string
}

func (g Generator) Generate(responsePayload []byte, outputPath string) error {
	data, err := g.GenerateBytes(responsePayload)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, data, 0o640); err != nil {
		return fmt.Errorf("save PDF: %w", err)
	}
	return nil
}

func (g Generator) GenerateBytes(responsePayload []byte) ([]byte, error) {
	if len(responsePayload) == 0 {
		return nil, fmt.Errorf("response payload is empty")
	}
	var root map[string]any
	if err := json.Unmarshal(responsePayload, &root); err != nil {
		return nil, fmt.Errorf("decode response payload: %w", err)
	}
	data := asMap(root["data"])
	if len(data) == 0 {
		return nil, fmt.Errorf("response payload does not contain data")
	}
	reports := validMaps(asSlice(data["reports"]))
	recommendationPages := 0
	if len(asSlice(data["common_recommendations"])) > 0 {
		recommendationPages = 1
	}
	insurancePages := 0
	if len(asMap(data["insurance_investment"])) > 0 {
		insurancePages = 1
	}
	totalPages := 2 + len(reports) + recommendationPages + insurancePages + 1

	p := fpdf.New("P", "mm", "A4", "")
	if err := registerHKGroteskFonts(p); err != nil {
		return nil, err
	}
	p.SetMargins(leftMargin, 18, rightMargin)
	p.SetAutoPageBreak(false, 0)
	p.SetTitle("AI Financial Plan Report", false)
	p.SetAuthor("BoltPe Financial Planner", false)

	pageNo := 0
	pageNo++
	g.cover(p, data, pageNo, totalPages)
	pageNo++
	g.summary(p, data, pageNo, totalPages)
	for _, report := range reports {
		pageNo++
		g.goalPage(p, report, pageNo, totalPages)
	}
	if recommendationPages == 1 {
		pageNo++
		g.recommendations(p, data, pageNo, totalPages)
	}
	if insurancePages == 1 {
		pageNo++
		g.insurance(p, data, pageNo, totalPages)
	}
	pageNo++
	g.disclaimer(p, data, pageNo, totalPages)

	if p.Error() != nil {
		return nil, fmt.Errorf("generate PDF: %w", p.Error())
	}
	var buffer bytes.Buffer
	if err := p.Output(&buffer); err != nil {
		return nil, fmt.Errorf("render PDF: %w", err)
	}
	return buffer.Bytes(), nil
}

func registerHKGroteskFonts(p *fpdf.Fpdf) error {
	regular, err := hkGroteskFonts.ReadFile("fonts/HKGrotesk-Regular.ttf")
	if err != nil {
		return fmt.Errorf("read HK Grotesk regular font: %w", err)
	}
	bold, err := hkGroteskFonts.ReadFile("fonts/HKGrotesk-Bold.ttf")
	if err != nil {
		return fmt.Errorf("read HK Grotesk bold font: %w", err)
	}

	p.AddUTF8FontFromBytes(hkGroteskFontFamily, "", regular)
	p.AddUTF8FontFromBytes(hkGroteskFontFamily, "B", bold)
	if p.Error() != nil {
		return fmt.Errorf("register HK Grotesk fonts: %w", p.Error())
	}
	return nil
}

func (g Generator) templateDir() string {
	if strings.TrimSpace(g.TemplateDir) != "" {
		return g.TemplateDir
	}
	if strings.TrimSpace(g.LogoPath) != "" {
		return filepath.Join(filepath.Dir(g.LogoPath), "pdf_templates")
	}
	return filepath.Join("assets", "pdf_templates")
}

func (g Generator) addTemplatePage(p *fpdf.Fpdf, templateNumber int, pageNo, totalPages int) {
	p.AddPage()
	p.SetFillColor(255, 255, 255)
	p.Rect(0, 0, pageWidth, pageHeight, "F")

	// BoltPe logo on every page.
	if strings.TrimSpace(g.LogoPath) != "" {
		if _, err := os.Stat(g.LogoPath); err == nil {
			p.ImageOptions(g.LogoPath, 170, 10, 20, 0, false, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
		}
	}

	// Footer.
	p.SetDrawColor(232, 232, 232)
	p.SetLineWidth(.2)
	p.Line(18, 284.5, 192, 284.5)
	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", 7)
	p.SetXY(leftMargin, 286.5)
	p.CellFormat(contentWidth, 4, fmt.Sprintf("Page %d of %d", pageNo, totalPages), "", 0, "C", false, 0, "")
}

func drawCoverBanner(p *fpdf.Fpdf, title, subtitle string, height float64) {
	// First page gradient:
	// Left  = #A34A10 = RGB(163, 74, 16)
	// Right = #652E0A = RGB(101, 46, 10)
	drawHorizontalGradient(
		p,
		0,
		22,
		pageWidth,
		height,
		163, 74, 16,
		101, 46, 10,
	)

	p.SetXY(18, 28)

	p.SetTextColor(255, 255, 255)

	p.SetFont(hkGroteskFontFamily, "B", 20)

	p.CellFormat(174, 10, safe(title), "", 1, "L", false, 0, "")

	if strings.TrimSpace(subtitle) != "" {

		p.SetX(18)
		// Subtitle
		p.SetFont(hkGroteskFontFamily, "", 14)
		p.CellFormat(174, 8, safe(subtitle), "", 0, "L", false, 0, "")

	}
}

func drawBanner(p *fpdf.Fpdf, title, subtitle string, height float64) {
	// Remaining pages:
	// #8F3B1A = RGB(143, 59, 26)
	p.SetFillColor(143, 59, 26)
	p.Rect(0, 22, pageWidth, height, "F")

	// Title
	p.SetXY(18, 27)
	p.SetTextColor(255, 255, 255)

	// Increase title size
	titleSize := fitTitleSize(title)

	if strings.EqualFold(strings.TrimSpace(title), "Summary") {
		titleSize = 26
	}

	p.SetFont(hkGroteskFontFamily, "B", titleSize)
	p.CellFormat(
		174,
		10,
		safe(title),
		"",
		1,
		"L",
		false,
		0,
		"",
	)

	// Subtitle
	if strings.TrimSpace(subtitle) != "" {
		p.SetX(18)

		// Keep subtitle same size
		p.SetFont(hkGroteskFontFamily, "", 14)

		p.CellFormat(
			174,
			6,
			safe(subtitle),
			"",
			0,
			"L",
			false,
			0,
			"",
		)
	}
}

func sectionTitle(p *fpdf.Fpdf, x, y float64, title string, size float64) {
	p.SetXY(x, y)
	p.SetTextColor(185, 76, 39)
	p.SetFont(hkGroteskFontFamily, "B", size)
	p.CellFormat(174, 7, safe(title), "", 1, "L", false, 0, "")
}

func (g Generator) cover(p *fpdf.Fpdf, d map[string]any, pageNo, totalPages int) {
	g.addTemplatePage(p, 1, pageNo, totalPages)
	drawCoverBanner(p, "Individual Financial Planning Report", "Full financial report generated through AI report", 40)
	if _, err := os.Stat(filepath.Join(filepath.Dir(g.LogoPath), "financial-report-icon.png")); err == nil {
		p.ImageOptions(filepath.Join(filepath.Dir(g.LogoPath), "financial-report-icon.png"), 164, 26, 25, 0, false, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	}
	customer := asMap(d["customer"])
	score := asMap(d["score"])

	// Generated date.
	p.SetXY(18, 45)
	p.SetTextColor(255, 255, 255)
	p.SetFont(hkGroteskFontFamily, "", 12.5)
	p.CellFormat(
		115,
		6,
		"Generated on "+safe(formatGeneratedAtIST(d["generated_at"])),
		"",
		0,
		"L",
		false,
		0,
		"",
	)

	// Personalised introduction and customer details. The greeting is measured
	// dynamically so long customer names never overlap the following sentence.
	name := safe(valueOr(customer["full_name"], "Customer"))
	greetingText := "Hi " + name + ","
	readyText := "your goal based Financial Plan is ready."
	introX := leftMargin
	introY := 66.0
	lineHeight := 7.0

	p.SetXY(introX, introY)
	p.SetFont(hkGroteskFontFamily, "B", 16)
	p.SetTextColor(185, 76, 39)
	greetingWidth := p.GetStringWidth(greetingText)

	p.SetFont(hkGroteskFontFamily, "", 14)
	readyWidth := p.GetStringWidth(" " + readyText)

	if greetingWidth+readyWidth <= contentWidth {
		// Normal case: keep the greeting and statement on the same line.
		p.SetFont(hkGroteskFontFamily, "B", 16)
		p.SetTextColor(185, 76, 39)
		p.CellFormat(greetingWidth, lineHeight, greetingText, "", 0, "L", false, 0, "")

		p.SetFont(hkGroteskFontFamily, "", 14)
		p.SetTextColor(0, 0, 0)
		p.CellFormat(contentWidth-greetingWidth, lineHeight, " "+readyText, "", 1, "L", false, 0, "")
	} else {
		// Long-name case: wrap the complete greeting first, then print the
		// statement on the next available line. MultiCell also handles names
		// wider than the entire content area.
		p.SetFont(hkGroteskFontFamily, "B", 16)
		p.SetTextColor(185, 76, 39)
		p.SetXY(introX, introY)
		p.MultiCell(contentWidth, lineHeight, greetingText, "", "L", false)

		p.SetX(introX)
		p.SetFont(hkGroteskFontFamily, "", 14)
		p.SetTextColor(0, 0, 0)
		p.MultiCell(contentWidth, lineHeight, readyText, "", "L", false)
	}

	// Position the description and the customer summary relative to the actual
	// greeting height so every following section moves down automatically.
	descriptionY := p.GetY() + 1.5
	p.SetXY(introX, descriptionY)
	p.SetFont(hkGroteskFontFamily, "", 12)
	p.SetTextColor(0, 0, 0)
	p.MultiCell(contentWidth, 5.5, "The report is organised by Instant Goals, Short Term Goals and Long Term Goals for a more professional advisory journey.", "", "L", false)

	dividerY := p.GetY() + 3.0
	p.SetDrawColor(220, 220, 220)
	p.Line(leftMargin, dividerY, pageWidth-rightMargin, dividerY)

	detailsLabelY := dividerY + 4.5
	detailsValueY := detailsLabelY + 6.0
	labels := []string{"Contact Number", "Age", "Financial Score", "Financial Status"}
	values := []string{valueOr(customer["contact"], "-"), valueOr(customer["age"], "-") + " Years", valueOr(score["value"], "-") + "/100", utils.FinancialStatusFromScore(scoreValueAsInt(score["value"]))}
	col := contentWidth / 4
	for i := range labels {
		x := leftMargin + float64(i)*col
		p.SetXY(x, detailsLabelY)
		p.SetTextColor(0, 0, 0)
		p.SetFont(hkGroteskFontFamily, "B", 12)
		p.CellFormat(col-2, 5, safe(labels[i]), "", 0, "L", false, 0, "")
		p.SetXY(x, detailsValueY)
		p.SetFont(hkGroteskFontFamily, "", 12)

		if labels[i] == "Financial Status" {

			// Width of heading
			p.SetFont(hkGroteskFontFamily, "B", 12)
			labelWidth := p.GetStringWidth(safe(labels[i]))

			// Width of value
			p.SetFont(hkGroteskFontFamily, "", 12)
			valueWidth := p.GetStringWidth(safe(values[i]))

			// Center value exactly below heading text
			valueX := x + (labelWidth-valueWidth)/2

			p.SetXY(valueX, detailsValueY)
			p.CellFormat(
				valueWidth,
				5,
				safe(values[i]),
				"",
				0,
				"L",
				false,
				0,
				"",
			)

		} else {

			p.CellFormat(
				col-1,
				5,
				safe(values[i]),
				"",
				0,
				"L",
				false,
				0,
				"",
			)
		}
	}

	// Recreate the original donut chart using live data.
	income, expenses, savings := cashflowValues(d)
	drawCashflowDonut(p, 105, 166, 42, income, expenses, savings)
}
func scoreValueAsInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(math.Round(v))
	case json.Number:
		n, err := strconv.Atoi(v.String())
		if err == nil {
			return n
		}
		f, _ := strconv.ParseFloat(v.String(), 64)
		return int(math.Round(f))
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return int(math.Round(f))
	default:
		return 0
	}
}

func (g Generator) summary(p *fpdf.Fpdf, d map[string]any, pageNo, totalPages int) {
	g.addTemplatePage(p, 2, pageNo, totalPages)
	drawBanner(p, "Summary", "Overview of selected goal types and current cash flow", 30)
	sectionTitle(p, 18, 61, "Current Position", 21)
	drawSquareBullets(p, summaryPointsWithRemainingFunds(d), 22, 72, 166, 13, 6.8)
	/*
	   // Goal Type Roadmap Banner
	   roadmapBannerY := 122.0
	   roadmapBannerH := 36.0

	   p.SetFillColor(brownR, brownG, brownB)
	   p.Rect(

	   	0,
	   	roadmapBannerY,
	   	pageWidth,
	   	roadmapBannerH,
	   	"F",

	   )

	   // Goal Type Roadmap - bigger title
	   p.SetXY(18, roadmapBannerY+5)
	   p.SetTextColor(255, 255, 255)
	   p.SetFont(hkGroteskFontFamily, "B", 26)

	   p.CellFormat(

	   	174,
	   	11,
	   	"Goal Type Roadmap",
	   	"",
	   	1,
	   	"L",
	   	false,
	   	0,
	   	"",

	   )

	   // Subtitle
	   p.SetX(18)
	   p.SetFont(hkGroteskFontFamily, "", 14)

	   p.CellFormat(

	   	174,
	   	8,
	   	"Most useful sequence to fund selected goals",
	   	"",
	   	0,
	   	"L",
	   	false,
	   	0,
	   	"",

	   )
	   sectionTitle(

	   	p,
	   	18,
	   	165,
	   	"Roadmap",
	   	21,

	   )

	   drawSquareBullets(

	   	p,
	   	asSlice(d["priority_roadmap"]),
	   	22,
	   	175,
	   	166,
	   	13.5,
	   	5.5,

	   )
	*/
}

func (g Generator) goalPage(
	p *fpdf.Fpdf,
	r map[string]any,
	pageNo, totalPages int,
) {
	g.addTemplatePage(p, 3, pageNo, totalPages)

	summary := asMap(r["summary"])
	goalType := strings.TrimSpace(valueOr(r["sub_title"], "Selected Goal"))
	goalName := strings.TrimSpace(valueOr(
		firstMapValue(summary, "Goal Name", "goal_name", "goalName"),
		goalType,
	))

	goalCategoryValue := strings.ToLower(strings.TrimSpace(valueOr(
		firstMapValue(summary, "Goal Type", "goal_type", "goalType"),
		"",
	)))

	var bannerTitle, bannerGoalType, bannerCategory string

	switch {
	case strings.Contains(goalCategoryValue, "instant"):
		purchaseName := strings.TrimSpace(goalName)

		if !strings.HasSuffix(strings.ToLower(purchaseName), " purchase") {
			purchaseName += " Purchase"
		}

		bannerTitle = "Instant Goal"
		bannerGoalType = purchaseName
		bannerCategory = "Instant Goals"

	case strings.Contains(goalCategoryValue, "short"):
		bannerTitle = "Short Term Goal"
		bannerGoalType = strings.TrimSpace(goalName)
		bannerCategory = "Short Term Goals"

	case strings.Contains(goalCategoryValue, "long"):
		bannerTitle = "Long Term Goal"
		bannerGoalType = strings.TrimSpace(goalName)
		bannerCategory = "Long Term Goals"

	default:
		bannerTitle = strings.TrimSuffix(goalType, " Planning") + " Financial Planning Report"
		bannerGoalType = goalType
	}

	drawBanner(
		p,
		bannerTitle,
		"Goal Name : "+bannerGoalType,
		30,
	)

	// Goal Category
	if bannerCategory != "" {
		p.SetXY(18, 51)
		p.SetTextColor(255, 255, 255)
		p.SetFont(hkGroteskFontFamily, "", 12)

		p.CellFormat(
			174,
			6,
			"Goal Category : "+bannerCategory,
			"",
			0,
			"L",
			false,
			0,
			"",
		)
	}

	// ---------------------------------------------------------
	// Goal Type Summary
	// ---------------------------------------------------------

	sectionTitle(
		p,
		18,
		56,
		"Goal Summary",
		18,
	)

	labels := []string{
		"Goal Category",
		"Goal Name",
		"Target Amount",
		"Target Horizon",
	}

	summaryGoalName := valueOr(
		firstMapValue(
			summary,
			"Goal Name",
			"goal_name",
			"goalName",
		),
		"-",
	)

	if strings.Contains(goalCategoryValue, "instant") &&
		summaryGoalName != "-" &&
		!strings.HasSuffix(
			strings.ToLower(strings.TrimSpace(summaryGoalName)),
			" purchase",
		) {
		summaryGoalName = strings.TrimSpace(summaryGoalName) + " Purchase"
	}

	values := []string{
		valueOr(
			firstMapValue(
				summary,
				"Goal Type",
				"goal_type",
				"goalType",
			),
			"-",
		),

		summaryGoalName,

		valueOr(
			firstMapValue(
				summary,
				"Target Amount / Today's Cost",
				"TargetAmount",
				"target_amount",
				"targetAmount",
			),
			"-",
		),

		valueOr(
			firstMapValue(
				summary,
				"Target Horizon",
				"target_horizon",
				"targetHorizon",
			),
			"-",
		),
	}

	rowY := []float64{
		64,
		72,
		80,
		88,
	}

	for i := range labels {
		y := rowY[i]

		// Left-side label
		p.SetXY(18, y)
		p.SetTextColor(0, 0, 0)
		p.SetFont(hkGroteskFontFamily, "", 12)

		p.CellFormat(
			88,
			6.5,
			safe(labels[i]),
			"",
			0,
			"L",
			false,
			0,
			"",
		)

		// Right-side value
		drawRightValue(
			p,
			112,
			y,
			80,
			6.5,
			values[i],
			12,
		)

		// Separator
		p.SetDrawColor(232, 232, 232)

		p.Line(
			18,
			y+5.5,
			192,
			y+5.5,
		)
	}

	// ---------------------------------------------------------
	// Goal Table
	// ---------------------------------------------------------

	rows := asSlice(r["table"])

	tableBottomY := 132.0

	if len(rows) > 0 {
		tableBottomY = drawGoalTableExact(
			p,
			asMap(rows[0]),
		)
	}

	// ---------------------------------------------------------
	// Feasibility Section
	// ---------------------------------------------------------

	analysisOffsetY := 0.0

	if tableBottomY > 132.0 {
		analysisOffsetY = tableBottomY - 132.0
	}

	analysisTitleY := 138.0 + analysisOffsetY
	analysisNoteY := 146.0 + analysisOffsetY

	feasibility := asMap(r["feasibility"])

	p.SetXY(
		18,
		analysisTitleY,
	)

	p.SetTextColor(
		185,
		76,
		39,
	)

	p.SetFont(
		hkGroteskFontFamily,
		"B",
		18,
	)

	p.CellFormat(
		174,
		7,
		strings.ToUpper(
			strings.ReplaceAll(
				safe(
					valueOr(
						feasibility["title"],
						"Goal Planning & Income Gap Analysis",
					),
				),
				"AI Feasibility & Income Gap Analysis",
				"Goal Planning & Income Gap Analysis",
			),
		),
		"",
		1,
		"L",
		false,
		0,
		"",
	)

	// ---------------------------------------------------------
	// Advisory Note
	// ---------------------------------------------------------

	p.SetXY(
		18,
		analysisNoteY,
	)

	p.SetTextColor(
		0,
		0,
		0,
	)

	p.SetFont(
		hkGroteskFontFamily,
		"",
		12,
	)

	advisoryNoteText := valueOr(
		firstMapValue(
			feasibility,
			"Boltpe Advisory Note",
			"AI Advisory Note",
			"ai_advisory_note",
			"advisory_note",
		),
		"-",
	)

	// Change only the sufficient-savings advisory note.
	if strings.Contains(
		strings.ToLower(advisoryNoteText),
		"current monthly savings are sufficient",
	) {
		advisoryNoteText =
			"Congratulations! Your current monthly savings are sufficient to achieve this goal within the selected timeline. Maintain your current savings plan to stay on track toward your goal."
	}

	p.MultiCell(
		174,
		6,
		"BoltPe Advisory Note: "+safe(advisoryNoteText),
		"",
		"L",
		false,
	)

	// For an Instant Goal that is already achievable within the
	// selected timeline, hide only the feasibility-detail grid.
	// The Assumptions section must still be shown so the customer
	// can see the inflation/return assumptions used for the goal.
	goalTypeForAdvisory := strings.ToLower(strings.TrimSpace(valueOr(
		firstMapValue(summary, "Goal Type", "goal_type", "goalType"),
		valueOr(firstMapValue(r, "type", "goal_type", "goalType"), ""),
	)))

	advisoryForDisplay := strings.ToLower(strings.TrimSpace(advisoryNoteText))
	instantGoalSavingsSufficient :=
		strings.Contains(goalTypeForAdvisory, "instant") &&
			strings.Contains(advisoryForDisplay, "current monthly savings are sufficient") &&
			strings.Contains(advisoryForDisplay, "selected timeline")

	// Do not return here. Even when an Instant Goal is feasible,
	// the Assumptions section is still required in the PDF.

	// ---------------------------------------------------------
	// Feasibility Details
	// ---------------------------------------------------------

	feasibilityFields := []struct {
		Label string
		Keys  []string
	}{
		{"Selected Timeline", []string{"Selected Timeline", "selected_timeline"}},
		{"Current Monthly Savings", []string{"Current Monthly Savings", "current_monthly_savings"}},
		{"Achievable Timeline With Current Savings", []string{"Achievable Timeline With Current Savings", "achievable_timeline_with_savings"}},
		{"Extra Time Required", []string{"Extra Time Required", "extra_time_required"}},
		{"Required Monthly Income To Keep Same Timeline", []string{"Required Monthly Income To Keep Same Timeline", "required_monthly_income"}},
	}

	advisoryNote := strings.ToLower(strings.TrimSpace(valueOr(
		firstMapValue(feasibility, "Boltpe Advisory Note", "AI Advisory Note", "ai_advisory_note", "advisory_note"),
		"",
	)))

	extraTimeRequired := strings.ToLower(strings.TrimSpace(valueOr(
		firstMapValue(feasibility, "Extra Time Required", "extra_time_required"),
		"",
	)))

	savingsSufficient :=
		strings.Contains(
			advisoryNote,
			"sufficient",
		) &&
			strings.Contains(
				advisoryNote,
				"selected timeline",
			)

	noExtraTimeRequired :=
		extraTimeRequired == "no extra time required" ||
			extraTimeRequired == "no extra time" ||
			extraTimeRequired == "0" ||
			extraTimeRequired == "0 months" ||
			extraTimeRequired == "0 years"

	if (savingsSufficient && noExtraTimeRequired) || instantGoalSavingsSufficient {
		// When the goal is already achievable in the selected timeline,
		// hide the feasibility detail grid. Assumptions are rendered below.
		feasibilityFields = nil
	}

	// Start dynamically after advisory note.
	// p.GetY() automatically reflects one-line or multi-line note height.
	y := p.GetY() + 4

	for _, field := range feasibilityFields {
		drawPairExact(
			p,
			18,
			y,
			174,
			field.Label,
			valueOr(firstMapValue(feasibility, field.Keys...), "-"),
			false,
		)

		y += 10
	}

	// Instant Goal behavior:
	// - Savings sufficient: hide feasibility detail rows, show Assumptions.
	// - Extra time required: show feasibility detail rows and Assumptions.

	// ---------------------------------------------------------
	// Assumptions
	// ---------------------------------------------------------

	assumptions := asMap(
		r["assumptions"],
	)

	// Dynamic position based on actual feasibility content above.
	//
	// If no feasibility rows:
	// Assumptions appears shortly after Advisory Note.
	//
	// If feasibility rows exist:
	// Assumptions appears after the last row.
	assumptionsTitleY := y + 6
	assumptionsRowsY := assumptionsTitleY + 12

	p.SetXY(
		18,
		assumptionsTitleY,
	)

	p.SetTextColor(
		185,
		76,
		39,
	)

	p.SetFont(
		hkGroteskFontFamily,
		"B",
		17,
	)

	p.CellFormat(
		174,
		7,
		"Assumptions",
		"",
		1,
		"L",
		false,
		0,
		"",
	)

	// Keep the assumptions section in the product-approved display order and
	// use user-facing labels rather than JSON/snake_case field names.
	assumptionFields := []struct {
		Label string
		Keys  []string
	}{
		{Label: "Expected Return", Keys: []string{"Expected Return", "expected_return"}},
		{Label: "Inflation Used", Keys: []string{"Inflation Used", "inflation_used"}},
		{Label: "Planning Style", Keys: []string{"Planning Style", "planning_style"}},
	}

	y = assumptionsRowsY
	for _, field := range assumptionFields {
		drawPairExact(
			p,
			18,
			y,
			174,
			field.Label,
			valueOr(firstMapValue(assumptions, field.Keys...), "-"),
			false,
		)
		y += 10
	}
}

func (g Generator) recommendations(p *fpdf.Fpdf, d map[string]any, pageNo, totalPages int) {
	g.addTemplatePage(p, 4, pageNo, totalPages)
	drawBanner(p, "BoltPe Priority Recommendations", "Common action points for all selected financial goals", 30)
	p.SetXY(18, 62)
	p.SetTextColor(185, 76, 39)
	p.SetFont(hkGroteskFontFamily, "B", 17)
	p.CellFormat(174, 8, "Advisor Action Plan", "", 1, "L", false, 0, "")
	drawSquareBullets(p, asSlice(d["common_recommendations"]), 22, 75, 166, 14, 7.2)
}

func (g Generator) insurance(
	p *fpdf.Fpdf,
	d map[string]any,
	pageNo, totalPages int,
) {
	g.addTemplatePage(p, 5, pageNo, totalPages)

	drawBanner(
		p,
		"BoltPe Recommended Investment",
		"Common action points for all selected financial goals",
		30,
	)

	ii := asMap(d["insurance_investment"])

	// ---------------------------------------------------------
	// Boltpe Family & Cashflow Interpretation
	// ---------------------------------------------------------

	p.SetXY(18, 53)
	p.SetTextColor(185, 76, 39)
	p.SetFont(hkGroteskFontFamily, "B", 18)
	p.CellFormat(
		174,
		7,
		"Family & Cashflow Interpretation",
		"",
		1,
		"L",
		false,
		0,
		"",
	)

	drawSquareBullets(
		p,
		asSlice(ii["interpretation"]),
		22,
		61,
		166,
		11,
		5.8,
	)

	// Get the ending Y position of the bullet list.
	protectionTitleY := p.GetY() + 4

	// Maintain minimum spacing for shorter lists.
	if protectionTitleY < 94 {
		protectionTitleY = 94
	}

	// ---------------------------------------------------------
	// Protection Gap Summary
	// ---------------------------------------------------------

	sectionTitle(
		p,
		18,
		protectionTitleY,
		"Protection Gap Summary",
		18,
	)

	gap := asMap(ii["gap_summary"])

	protectionRows := []struct {
		label string
		value string
	}{
		{
			label: "Total Recommended Cover",
			value: valueOr(gap["Total Recommended Cover"], "-"),
		},
		{
			label: "Total Existing Cover",
			value: valueOr(gap["Total Existing Cover"], "-"),
		},
		{
			label: "Total Protection Gap",
			value: valueOr(gap["Total Protection Gap"], "-"),
		},
		{
			label: "Gap Status",
			value: valueOr(gap["Gap Status"], "-"),
		},
	}

	y := protectionTitleY + 8

	for _, row := range protectionRows {
		drawProtectionSummaryRow(
			p,
			18,
			y,
			174,
			row.label,
			row.value,
		)

		y += 7
	}

	// ---------------------------------------------------------
	// Boltpe Recommended Investments Allocation
	// ---------------------------------------------------------

	investmentTitleY := y + 4

	sectionTitle(
		p,
		18,
		investmentTitleY,
		"BoltPe Recommended Investments Allocation",
		18,
	)

	p.SetXY(18, investmentTitleY+8)
	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", 12)
	p.CellFormat(
		174,
		5,
		"Goal type based monthly surplus allocation",
		"",
		0,
		"L",
		false,
		0,
		"",
	)

	rows := asSlice(ii["investment_rows"])
	investmentTableY := investmentTitleY + 16

	drawInvestmentTableExact(
		p,
		rows,
		18,
		investmentTableY,
	)

	// Table header = 14 mm
	// Each data row = 13 mm
	investmentTableEndY :=
		investmentTableY +
			14 +
			float64(len(rows))*13

	// ---------------------------------------------------------
	// Suggested Asset Allocation
	// ---------------------------------------------------------

	assetTitleY := investmentTableEndY + 7

	p.SetXY(18, assetTitleY)
	p.SetTextColor(185, 76, 39)
	p.SetFont(hkGroteskFontFamily, "B", 18)
	p.CellFormat(
		174,
		7,
		"Suggested Asset Allocation",
		"",
		1,
		"L",
		false,
		0,
		"",
	)

	allocation := asMap(ii["asset_allocation"])

	ordered := []string{
		"Equity",
		"Debt",
		"Gold / Alternatives",
	}

	y = assetTitleY + 9

	for _, key := range ordered {
		drawPairExact(
			p,
			18,
			y,
			174,
			key,
			valueOr(allocation[key], "-"),
			false,
		)

		y += 8
	}
}

func (g Generator) disclaimer(
	p *fpdf.Fpdf,
	d map[string]any,
	pageNo, totalPages int,
) {
	g.addTemplatePage(p, 6, pageNo, totalPages)

	drawBanner(p, "Email Confirmation", "", 30)

	// Disclaimer title moved upward to remove unwanted empty space.
	sectionTitle(p, 18, 58, "Disclaimer", 21)

	p.SetXY(18, 70)
	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", 12)

	p.MultiCell(
		174,
		5.5,
		`This report is generated for educational and advisory purposes based on the information provided by the user. It should not be treated as guaranteed investment return, legal advice, tax advice, or a final financial product recommendation.

Market-linked investments are subject to market risks. Insurance and investment suitability may vary based on underwriting, financial condition, risk profile, product terms, and regulatory guidelines. Please consult a qualified financial advisor before making any investment or insurance decision.`,
		"",
		"L",
		false,
	)

	// Keep the same content and increase only its font size.
	p.SetXY(18, 140)
	p.SetTextColor(185, 76, 39)
	p.SetFont(hkGroteskFontFamily, "B", 21)
	p.CellFormat(
		174,
		10,
		"All The Best, Thank You for Choosing BoltPe!",
		"",
		1,
		"L",
		false,
		0,
		"",
	)
}

func drawGoalTableExact(p *fpdf.Fpdf, row map[string]any) float64 {
	goalType := strings.ToLower(strings.TrimSpace(valueOr(
		firstMapValue(row, "Goal Type", "goal_type", "goalType"),
		"",
	)))
	isInstantGoal := strings.Contains(goalType, "instant")
	headers := []string{
		"Goal",
		"Goal Type",
		"Years / Months Left",
		"Future Corpus Needed",
		"Monthly Saving / SIP Required",
		"Lumpsum Needed Today",
	}

	// Use the same six-column calculation table for Instant, Short-Term,
	// and Long-Term goals. Instant goals also calculate Future Corpus and
	// Lumpsum Needed Today, so those values must be visible in the PDF.
	widths := []float64{29, 29, 29, 29, 30, 28}

	x := 18.0
	y := 98.0

	headerH := 18.0
	rowH := 16.0

	// Increase table height only for Instant Goals.
	if isInstantGoal {
		headerH = 24.0
		rowH = 20.0
	}

	goalName := strings.TrimSpace(valueOr(firstMapValue(row, "Goal", "goal"), "-"))

	// Increase table row height only for these two long goal names.
	switch strings.ToLower(goalName) {
	case strings.ToLower("Higher Education"),
		strings.ToLower("Home Purchase / Home Upgrade"):
		rowH = 24.0
	}

	whiteRect(
		p,
		18,
		y,
		174,
		headerH+rowH+1,
	)

	p.SetFillColor(orangeR, orangeG, orangeB)
	p.SetTextColor(255, 255, 255)
	p.SetFont(hkGroteskFontFamily, "", 8.5)

	for i, header := range headers {
		tableCell(
			p,
			x,
			y,
			widths[i],
			headerH,
			header,
			true,
			12,
			true,
		)

		x += widths[i]
	}

	x = 18.0

	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", 8.5)

	for i, header := range headers {
		var cellValue string
		switch header {
		case "Goal":
			cellValue = valueOr(firstMapValue(row, "Goal", "goal"), "-")
		case "Goal Type":
			cellValue = valueOr(firstMapValue(row, "Goal Type", "goal_type", "goalType"), "-")
		case "Years / Months Left":
			cellValue = valueOr(firstMapValue(row, "Years / Months Left", "years_months_left"), "-")
		case "Future Corpus Needed":
			cellValue = valueOr(firstMapValue(row, "Future Corpus Needed", "future_corpus_needed"), "-")
		case "Monthly Saving / SIP Required":
			cellValue = valueOr(firstMapValue(row, "Monthly Saving / SIP Required", "monthly_saving_required"), "-")
		case "Lumpsum Needed Today":
			cellValue = valueOr(firstMapValue(row, "Lumpsum Needed Today", "lumpsum_needed_today"), "-")
		default:
			cellValue = "-"
		}

		tableCell(
			p,
			x,
			y+headerH,
			widths[i],
			rowH,
			cellValue,
			false,
			12,
			false,
		)

		x += widths[i]
	}

	return y + headerH + rowH
}

func drawInvestmentTableExact(p *fpdf.Fpdf, rows []any, x0, y0 float64) {
	headers := []string{"Investment Bucket", "Suggested Product Type", "Monthly Allocation", "Purpose"}
	widths := []float64{44, 54, 43, 33}
	x := x0
	p.SetFillColor(orangeR, orangeG, orangeB)
	p.SetTextColor(255, 255, 255)
	for i, h := range headers {
		tableCell(p, x, y0, widths[i], 14, h, true, 12, true)
		x += widths[i]
	}
	y := y0 + 14
	for _, v := range rows {
		row := asMap(v)
		x = x0
		p.SetTextColor(0, 0, 0)
		for i, h := range headers {
			tableCell(p, x, y, widths[i], 13, valueOr(row[h], "-"), false, 11, false)
			x += widths[i]
		}
		y += 13
	}
}

func tableCell(p *fpdf.Fpdf, x, y, w, h float64, text string, fill bool, size float64, bold bool) {
	p.SetDrawColor(0, 0, 0)
	p.SetLineWidth(.2)
	if fill {
		p.Rect(x, y, w, h, "F")
	}
	p.Rect(x, y, w, h, "D")
	p.SetXY(x+2, y+2)
	fontStyle := ""
	if bold {
		fontStyle = "B"
	}
	p.SetFont(hkGroteskFontFamily, fontStyle, size)
	p.MultiCell(w-4, 4, safeCurrency(text), "", "L", false)
}

func drawPairExact(
	p *fpdf.Fpdf,
	x, y, w float64,
	key, value string,
	bold bool,
) {
	p.SetDrawColor(224, 224, 224)

	// Increased row height for 14pt font
	p.Line(x, y+8.5, x+w, y+8.5)

	p.SetXY(x, y)
	p.SetTextColor(0, 0, 0)

	if bold {
		p.SetFont(hkGroteskFontFamily, "B", 11)
	} else {
		p.SetFont(hkGroteskFontFamily, "", 11)
	}

	p.CellFormat(
		w*.58,
		8,
		safe(key),
		"",
		0,
		"L",
		false,
		0,
		"",
	)

	drawRightValue(
		p,
		x+w*.58,
		y,
		w*.42,
		8,
		value,
		11,
	)
}

func drawRightValue(p *fpdf.Fpdf, x, y, w, h float64, value string, size float64) {
	p.SetXY(x, y)
	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", size)
	p.CellFormat(w, h, safeCurrency(value), "", 0, "R", false, 0, "")
}

func drawSquareBullets(p *fpdf.Fpdf, values []any, x, y, w, size, lineHeight float64) {
	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", size)
	for _, v := range values {
		text := strings.TrimSpace(safeCurrency(fmt.Sprint(v)))
		if text == "" {
			continue
		}
		p.SetFillColor(25, 25, 25)
		p.Rect(x, y+1.7, 1.3, 1.3, "F")
		p.SetXY(x+4, y)
		p.MultiCell(w-4, lineHeight, text, "", "L", false)
		y = p.GetY() + .7
	}
}

func drawCashflowDonut(p *fpdf.Fpdf, cx, cy, radius, income, expenses, savings float64) {
	if income <= 0 {
		income = expenses + savings
	}
	if income <= 0 {
		income = 1
	}
	if expenses < 0 {
		expenses = 0
	}
	if savings < 0 {
		savings = 0
	}

	remaining := income - expenses - savings
	if remaining < 0 {
		remaining = 0
	}

	total := expenses + savings + remaining
	if total <= 0 {
		expenses = income * .6
		savings = income * .3
		remaining = income - expenses - savings
		total = income
	}

	expAngle := 360 * expenses / total
	savingsAngle := 360 * savings / total
	expEnd := -90 + expAngle
	savingsEnd := expEnd + savingsAngle

	drawSector(p, cx, cy, radius, -90, expEnd, 35, 104, 185)
	drawSector(p, cx, cy, radius, expEnd, savingsEnd, 245, 180, 0)
	if remaining > 0 {
		drawSector(p, cx, cy, radius, savingsEnd, 270, 220, 220, 220)
	}

	p.SetFillColor(255, 255, 255)
	p.Circle(cx, cy, radius*.60, "F")

	separatorAngles := []float64{-90, expEnd}
	if remaining > 0 {
		separatorAngles = append(separatorAngles, savingsEnd)
	}
	for _, angle := range separatorAngles {
		rad := angle * math.Pi / 180
		p.SetDrawColor(255, 255, 255)
		p.SetLineWidth(2.2)
		p.Line(
			cx+radius*.57*math.Cos(rad),
			cy+radius*.57*math.Sin(rad),
			cx+radius*1.02*math.Cos(rad),
			cy+radius*1.02*math.Sin(rad),
		)
	}

	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", 12)
	p.SetXY(cx-30, cy-8)
	p.CellFormat(60, 6, "Monthly Income", "", 1, "C", false, 0, "")
	p.SetFont(hkGroteskFontFamily, "", 18)
	p.SetX(cx - 30)
	p.CellFormat(60, 9, formatIndianAmount(income), "", 1, "C", false, 0, "")

	ly := cy + radius + 15

	legendItems := []struct {
		x                float64
		label            string
		amount           float64
		red, green, blue int
	}{
		{cx - 78, "Monthly Expenses", expenses, 35, 104, 185},
		{cx - 20, "Monthly Savings", savings, 245, 180, 0},
		{cx + 38, "Remaining Funds", remaining, 220, 220, 220},
	}

	for _, item := range legendItems {
		p.SetFillColor(item.red, item.green, item.blue)

		// Bigger colored box
		p.RoundedRect(item.x, ly, 8, 7, 1.5, "1234", "F")

		// Label
		p.SetXY(item.x+11, ly-1)
		p.SetFont(hkGroteskFontFamily, "", 14)
		p.CellFormat(48, 7, item.label, "", 1, "L", false, 0, "")

		// Amount
		p.SetX(item.x + 11)
		p.SetFont(hkGroteskFontFamily, "", 14)
		p.CellFormat(
			48,
			7,
			"Rs. "+formatIndianAmount(item.amount),
			"",
			1,
			"L",
			false,
			0,
			"",
		)
	}
}

func drawSector(p *fpdf.Fpdf, cx, cy, radius, startDeg, endDeg float64, red, green, blue int) {
	points := []fpdf.PointType{{X: cx, Y: cy}}
	steps := int(math.Max(12, math.Abs(endDeg-startDeg)/3))
	for i := 0; i <= steps; i++ {
		a := startDeg + (endDeg-startDeg)*float64(i)/float64(steps)
		rad := a * math.Pi / 180
		points = append(points, fpdf.PointType{X: cx + radius*math.Cos(rad), Y: cy + radius*math.Sin(rad)})
	}
	p.SetFillColor(red, green, blue)
	p.Polygon(points, "F")
}

func whiteRect(p *fpdf.Fpdf, x, y, w, h float64) {
	p.SetFillColor(255, 255, 255)
	p.Rect(x, y, w, h, "F")
}

var amountPattern = regexp.MustCompile(`(?:Rs\.?|₹)?\s*([0-9][0-9,]*(?:\.[0-9]+)?)`)

func summaryPointsWithRemainingFunds(d map[string]any) []any {
	points := asSlice(asMap(d["summary"])["points"])
	if len(points) == 0 {
		return points
	}
	income, expenses, savings := cashflowValues(d)
	remaining := income - expenses - savings
	if remaining < 0 {
		remaining = 0
	}
	updated := make([]any, 0, len(points))
	for _, value := range points {
		text := strings.TrimSpace(fmt.Sprint(value))
		lower := strings.ToLower(text)
		// Add remaining funds only to the cash-flow line that already contains
		// both monthly expenses and monthly savings.
		if strings.Contains(lower, "monthly expenses") &&
			strings.Contains(lower, "monthly savings") &&
			!strings.Contains(lower, "remaining funds") {
			text = strings.TrimSuffix(text, ".")
			text = fmt.Sprintf(
				"%s, and remaining funds are Rs. %s.",
				text,
				formatIndianAmount(remaining),
			)
		}
		updated = append(updated, text)
	}
	return updated
}
func cashflowValues(d map[string]any) (income, expenses, savings float64) {
	// The structured BoltPe response carries the source cash-flow values in
	// customer. Read those typed response fields first instead of deriving
	// financial data by scraping human-readable summary sentences.
	customer := asMap(d["customer"])
	income = amountFromAny(firstMapValue(customer, "monthly_income", "monthlyIncome", "Monthly Income"))
	expenses = amountFromAny(firstMapValue(customer, "monthly_expenses", "monthlyExpenses", "Monthly Expenses"))
	savings = amountFromAny(firstMapValue(customer, "monthly_savings", "monthlySavings", "Monthly Savings"))

	if income > 0 || expenses > 0 || savings > 0 {
		return
	}

	// Backward-compatible fallback for older stored reports that did not keep
	// the cash-flow values in customer.
	for _, value := range asSlice(asMap(d["summary"])["points"]) {
		text := strings.ToLower(fmt.Sprint(value))
		matches := amountPattern.FindAllStringSubmatch(text, -1)
		if strings.Contains(text, "monthly income") && len(matches) > 0 {
			income = parseAmount(matches[0][1])
		}
		if strings.Contains(text, "monthly expenses") && len(matches) > 0 {
			expenses = parseAmount(matches[0][1])
			if len(matches) > 1 {
				savings = parseAmount(matches[1][1])
			}
		}
	}
	return
}

func amountFromAny(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	text := strings.TrimSpace(fmt.Sprint(v))
	text = strings.ReplaceAll(text, "₹", "")
	text = strings.ReplaceAll(text, "Rs.", "")
	text = strings.ReplaceAll(text, "Rs", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.TrimSpace(text)
	f, _ := strconv.ParseFloat(text, 64)
	return f
}
func parseAmount(v string) float64 {
	v = strings.ReplaceAll(v, ",", "")
	n, _ := strconv.ParseFloat(v, 64)
	return n
}
func formatIndianAmount(value float64) string {
	n := int64(math.Round(value))
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) > 3 {
		last := s[len(s)-3:]
		rest := s[:len(s)-3]
		groups := []string{}
		for len(rest) > 2 {
			groups = append([]string{rest[len(rest)-2:]}, groups...)
			rest = rest[:len(rest)-2]
		}
		if rest != "" {
			groups = append([]string{rest}, groups...)
		}
		s = strings.Join(groups, ",") + "," + last
	}
	if neg {
		return "-" + s
	}
	return s
}

func fitTitleSize(title string) float64 {
	switch {
	case len(title) > 58:
		return 16
	case len(title) > 45:
		return 18
	case len(title) > 34:
		return 20
	default:
		return 22
	}
}
func safe(v string) string {
	return strings.NewReplacer("₹", "Rs. ", "•", "-", "–", "-", "—", "-", "’", "'", "“", "\"", "”", "\"").Replace(v)
}
func safeCurrency(v string) string { return safe(v) }
func firstMapValue(data map[string]any, keys ...string) any {
	for _, key := range keys {
		value, exists := data[key]
		if !exists || value == nil {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			return value
		}
	}
	return nil
}
func valueOr(v any, fallback string) string {
	if v == nil {
		return fallback
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return fallback
	}
	return s
}
func asMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return m
}
func asSlice(v any) []any {
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	return s
}
func validMaps(values []any) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for _, v := range values {
		m := asMap(v)
		if len(m) > 0 {
			out = append(out, m)
		}
	}
	return out
}

func drawHorizontalGradient(
	p *fpdf.Fpdf,
	x, y, width, height float64,
	startR, startG, startB int,
	endR, endG, endB int,
) {
	const steps = 200

	stepWidth := width / float64(steps)

	for i := 0; i < steps; i++ {
		ratio := float64(i) / float64(steps-1)

		r := int(float64(startR) + float64(endR-startR)*ratio)
		g := int(float64(startG) + float64(endG-startG)*ratio)
		b := int(float64(startB) + float64(endB-startB)*ratio)

		p.SetFillColor(r, g, b)
		p.Rect(
			x+float64(i)*stepWidth,
			y,
			stepWidth+0.2,
			height,
			"F",
		)
	}
}

func drawProtectionSummaryRow(
	p *fpdf.Fpdf,
	x, y, width float64,
	label, value string,
) {
	rowHeight := 8.0

	// Left-side label.
	p.SetXY(x, y)
	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", 11)
	p.CellFormat(
		width*0.62,
		rowHeight,
		safe(label),
		"",
		0,
		"L",
		false,
		0,
		"",
	)

	// Right-side value.
	p.SetXY(x+(width*0.62), y)
	p.SetTextColor(0, 0, 0)
	p.SetFont(hkGroteskFontFamily, "", 11)
	p.CellFormat(
		width*0.38,
		rowHeight,
		safeCurrency(value),
		"",
		0,
		"R",
		false,
		0,
		"",
	)

	// Bottom separator line.
	p.SetDrawColor(224, 224, 224)
	p.SetLineWidth(0.2)
	p.Line(
		x,
		y+rowHeight,
		x+width,
		y+rowHeight,
	)
}

func financialStatus(scoreValue float64) string {

	switch {
	case scoreValue >= 75:
		return "High"
	case scoreValue >= 50:
		return "Average"
	default:
		return "Low"
	}

}

func formatGeneratedAtIST(value any) string {
	raw := strings.TrimSpace(fmt.Sprint(value))
	if raw == "" || raw == "<nil>" || raw == "-" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	ist := time.FixedZone("IST", 5*60*60+30*60)
	return t.In(ist).Format("02/01/2006 03:04:05 PM MST")
}
