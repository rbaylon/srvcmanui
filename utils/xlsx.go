package utils

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
)

// colName turns a zero-based column index into its spreadsheet letters
// (0 -> A, 25 -> Z, 26 -> AA).
func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

func xmlEsc(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

// BuildXLSX writes a single-sheet Excel workbook using only the standard
// library. Cells may be string, int, int64, uint, float64 or nil (empty);
// strings are stored as inline text, so a value such as "=1+1" stays text
// and is never evaluated by Excel. The header row is bold and frozen.
func BuildXLSX(sheet string, header []string, rows [][]any) ([]byte, error) {
	var sb bytes.Buffer
	sb.WriteString(xml.Header)
	sb.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	sb.WriteString(`<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`)
	sb.WriteString(`<cols>`)
	for i := range header {
		fmt.Fprintf(&sb, `<col min="%d" max="%d" width="16" customWidth="1"/>`, i+1, i+1)
	}
	sb.WriteString(`</cols><sheetData>`)

	sb.WriteString(`<row r="1">`)
	for i, h := range header {
		fmt.Fprintf(&sb, `<c r="%s1" s="1" t="inlineStr"><is><t>%s</t></is></c>`, colName(i), xmlEsc(h))
	}
	sb.WriteString(`</row>`)

	for r, row := range rows {
		fmt.Fprintf(&sb, `<row r="%d">`, r+2)
		for c, v := range row {
			ref := colName(c) + strconv.Itoa(r+2)
			switch x := v.(type) {
			case nil:
			case string:
				if x != "" {
					fmt.Fprintf(&sb, `<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, xmlEsc(x))
				}
			case int:
				fmt.Fprintf(&sb, `<c r="%s"><v>%d</v></c>`, ref, x)
			case int64:
				fmt.Fprintf(&sb, `<c r="%s"><v>%d</v></c>`, ref, x)
			case uint:
				fmt.Fprintf(&sb, `<c r="%s"><v>%d</v></c>`, ref, x)
			case float64:
				fmt.Fprintf(&sb, `<c r="%s"><v>%s</v></c>`, ref, strconv.FormatFloat(x, 'f', -1, 64))
			default:
				return nil, fmt.Errorf("xlsx: unsupported cell type %T", v)
			}
		}
		sb.WriteString(`</row>`)
	}
	sb.WriteString(`</sheetData></worksheet>`)

	files := []struct{ name, body string }{
		{"[Content_Types].xml", xml.Header + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
			`</Types>`},
		{"_rels/.rels", xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`},
		{"xl/workbook.xml", xml.Header + `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			`<sheets><sheet name="` + xmlEsc(sheet) + `" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
			`</Relationships>`},
		{"xl/styles.xml", xml.Header + `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>` +
			`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
			`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
			`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
			`<cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
			`<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs>` +
			`</styleSheet>`},
		{"xl/worksheets/sheet1.xml", sb.String()},
	}

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(f.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// ServeXLSX sends a workbook as a file download.
func ServeXLSX(w http.ResponseWriter, filename string, data []byte) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}
