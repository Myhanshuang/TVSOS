"use strict";
const fs = require("fs");
const path = require("path");
const {
  Document, Packer, Paragraph, TextRun, Table, TableRow, TableCell,
  Header, Footer, AlignmentType, HeadingLevel, WidthType, BorderStyle,
  TableOfContents, NumberFormat, PageNumber,
} = require("docx");
const H = require("./docx_helpers.js");
const ch1 = require("./docx_ch1.js");
const ch23 = require("./docx_ch23.js");

const OUTPUT = process.argv[2] ? path.resolve(process.argv[2]) : path.join(__dirname, "..", "综合设计III中期报告.docx");

// 注：正式报告的封面已由用户在 WPS 中替换为整页 EMF 官方封面（含边框形状）。
// 本脚本再生成时仍是文本封面；如需与正式版一致，请以已交付 docx 为准做 XML 级合并（见 _surgery.py）。
// ================= cover =================
var cover = [];
cover.push(H.spacer(1600));
cover.push(H.coverTitle("信息与软件工程学院", 44));
cover.push(H.coverTitle("综合设计 III 中期报告", 44, 200));
cover.push(H.spacer(800));
cover.push(H.coverLine("课程名称", "进阶式挑战性综合项目 III（智能交通）"));
cover.push(H.coverLine("课题名称", "运输车辆调度优化与仿真"));
cover.push(H.coverLine("指导教师", "李贞昊"));
cover.push(H.coverLine("所在系别", "软件工程"));
cover.push(H.coverLine("执行学期", "第五学期"));
cover.push(H.spacer(400));
cover.push(new Paragraph({
  alignment: AlignmentType.CENTER,
  spacing: { after: 120 },
  children: [new TextRun({ text: "学生信息：", font: H.FONT_HEAD, size: 28, bold: true })],
}));
(function () {
  const students = [
    ["1（组长）", "2024090904002", "胡杨"],
    ["2", "2024090910005", "丁宁"],
    ["3", "2024090906027", "胡天琦"],
    ["4", "2024090910008", "华观中"],
    ["5", "2024090910020", "万飞翔"],
    ["6", "2024090906006", "李韦成"],
  ];
  const rows = [
    ["序号", "学号", "姓名"].map(function (h) {
      return { lines: h, align: AlignmentType.CENTER };
    }),
  ];
  students.forEach(function (s) {
    rows.push(s.map(function (v) {
      return { lines: v, align: AlignmentType.CENTER };
    }));
  });
  cover.push(new Table({
    width: { size: 62, type: WidthType.PERCENTAGE },
    alignment: AlignmentType.CENTER,
    borders: {
      top: { style: BorderStyle.SINGLE, size: 8, color: "000000" },
      bottom: { style: BorderStyle.SINGLE, size: 8, color: "000000" },
      left: H.NO_BORDER, right: H.NO_BORDER,
      insideHorizontal: H.NO_BORDER, insideVertical: H.NO_BORDER,
    },
    rows: rows.map(function (r, i) {
      return new TableRow({
        children: r.map(function (c) {
          const o = Object.assign({}, c);
          if (i === 0) {
            o.bold = true;
            o.borders = { top: H.NO_BORDER, bottom: H.THIN, left: H.NO_BORDER, right: H.NO_BORDER };
          }
          return H.cellP(o.lines, o);
        }),
      });
    }),
  }));
})();

// ================= TOC =================
var tocSection = [];
tocSection.push(new Paragraph({
  alignment: AlignmentType.CENTER,
  spacing: { before: 240, after: 300 },
  children: [new TextRun({ text: "目  录", font: H.FONT_HEAD, size: 32, bold: true })],
}));
tocSection.push(new TableOfContents("目录", { hyperlink: true, headingStyleRange: "1-3" }));

// ================= headers / footers =================
var emptyHeader = { default: new Header({ children: [] }) };
var emptyFooter = { default: new Footer({ children: [] }) };
var mainHeader = {
  default: new Header({
    children: [new Paragraph({
      alignment: AlignmentType.CENTER,
      border: { bottom: { style: BorderStyle.SINGLE, size: 4, color: "999999" } },
      children: [new TextRun({ text: "信软学院综合设计 III 中期报告", font: H.FONT_BODY, size: 18, color: "666666" })],
    })],
  }),
};
var mainFooter = {
  default: new Footer({
    children: [new Paragraph({
      alignment: AlignmentType.CENTER,
      children: [new TextRun({
        children: ["第 ", PageNumber.CURRENT, " 页，共 ", PageNumber.TOTAL_PAGES, " 页"],
        font: H.FONT_BODY, size: 18,
      })],
    })],
  }),
};
var PAGE = {
  size: { width: 11906, height: 16838 },
  margin: { top: 1440, bottom: 1440, left: 1800, right: 1800 },
};

var doc = new Document({
  features: { updateFields: true },
  creator: "TVSOS 小组",
  title: "综合设计 III 中期报告——运输车辆调度优化与仿真",
  styles: {
    default: {
      document: { run: { font: H.FONT_BODY, size: 24 } },
    },
  },
  sections: [
    {
      properties: { page: Object.assign({}, PAGE) },
      headers: emptyHeader,
      footers: emptyFooter,
      children: cover,
    },
    {
      properties: { page: Object.assign({}, PAGE, { pageNumbers: { start: 1, formatType: NumberFormat.LOWER_ROMAN } }) },
      headers: emptyHeader,
      footers: emptyFooter,
      children: tocSection,
    },
    {
      properties: { page: Object.assign({}, PAGE, { pageNumbers: { start: 1, formatType: NumberFormat.DECIMAL } }) },
      headers: mainHeader,
      footers: mainFooter,
      children: ch1.concat(ch23),
    },
  ],
});

Packer.toBuffer(doc).then(function (buffer) {
  fs.writeFileSync(OUTPUT, buffer);
  console.log("WROTE", OUTPUT, buffer.length, "bytes");
}).catch(function (e) {
  console.error("BUILD FAILED:", (e && e.stack) || e);
  process.exit(1);
});
