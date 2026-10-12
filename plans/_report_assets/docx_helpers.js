"use strict";
const fs = require("fs");
const path = require("path");
const {
  Paragraph, TextRun, Table, TableRow, TableCell, ImageRun,
  Header, Footer, AlignmentType, HeadingLevel, WidthType,
  BorderStyle, ShadingType, VerticalAlign, NumberFormat, PageNumber,
} = require("docx");

const FONT_BODY = { ascii: "Times New Roman", eastAsia: "SimSun", hAnsi: "Times New Roman" };
const FONT_HEAD = { ascii: "Times New Roman", eastAsia: "SimHei", hAnsi: "Times New Roman" };
const FONT_CODE = { ascii: "Consolas", eastAsia: "SimSun", hAnsi: "Consolas" };

function readPng(file) {
  const buf = fs.readFileSync(path.join(__dirname, file));
  return { buf: buf, w: buf.readUInt32BE(16), h: buf.readUInt32BE(20) };
}
function figure(file, targetW) {
  const p = readPng(file);
  const width = targetW || 520;
  const height = Math.round(p.h * width / p.w);
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { before: 160, after: 60 },
    children: [new ImageRun({ type: "png", data: p.buf, transformation: { width: width, height: height } })],
  });
}
function caption(text) {
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { after: 240 },
    children: [new TextRun({ text: text, font: FONT_HEAD, size: 21, bold: true })],
  });
}
function codeCaption(text) {
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { before: 160, after: 80 },
    children: [new TextRun({ text: text, font: FONT_HEAD, size: 21, bold: true })],
  });
}
function t(text) { return { t: text, b: false }; }
function b(text) { return { t: text, b: true }; }
function P(parts, opts) {
  opts = opts || {};
  if (typeof parts === "string") parts = [t(parts)];
  return new Paragraph({
    alignment: AlignmentType.JUSTIFIED,
    spacing: { line: 340, lineRule: "auto" },
    indent: opts.noIndent ? undefined : { firstLine: 480 },
    children: parts.map(function (r) {
      return new TextRun({ text: r.t, bold: r.b, font: FONT_BODY, size: 24 });
    }),
  });
}
function H1(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_1,
    pageBreakBefore: true,
    alignment: AlignmentType.CENTER,
    spacing: { before: 240, after: 360 },
    children: [new TextRun({ text: text, font: FONT_HEAD, size: 32, bold: true, color: "000000" })],
  });
}
function H2(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_2,
    spacing: { before: 320, after: 200 },
    children: [new TextRun({ text: text, font: FONT_HEAD, size: 28, bold: true, color: "000000" })],
  });
}
function H3(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_3,
    spacing: { before: 240, after: 160 },
    children: [new TextRun({ text: text, font: FONT_HEAD, size: 24, bold: true, color: "000000" })],
  });
}
function codeBlock(lines) {
  return lines.map(function (ln) {
    return new Paragraph({
      spacing: { line: 260, lineRule: "auto" },
      shading: { type: ShadingType.CLEAR, fill: "F2F2EC" },
      indent: { left: 420, right: 420 },
      children: [new TextRun({ text: ln.length ? ln : " ", font: FONT_CODE, size: 19 })],
    });
  });
}
function refP(idx, text) {
  return new Paragraph({
    alignment: AlignmentType.LEFT,
    spacing: { line: 320, lineRule: "auto", after: 60 },
    indent: { left: 480, hanging: 480 },
    children: [new TextRun({ text: "[" + idx + "]  " + text, font: FONT_BODY, size: 21 })],
  });
}
function spacer(h) {
  return new Paragraph({
    spacing: { before: 0, after: h },
    children: [new TextRun({ text: "", font: FONT_BODY, size: 24 })],
  });
}
function coverLine(label, value) {
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { line: 400, lineRule: "auto", before: 100, after: 100 },
    children: [
      new TextRun({ text: label + "：　", font: FONT_HEAD, size: 28, bold: true }),
      new TextRun({ text: "　" + value + "　", font: FONT_BODY, size: 28, underline: {} }),
    ],
  });
}
function coverTitle(text, size, before) {
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { before: before || 0, after: 120 },
    children: [new TextRun({ text: text, font: FONT_HEAD, size: size, bold: true })],
  });
}
const NO_BORDER = { style: BorderStyle.NONE, size: 0, color: "FFFFFF" };
const THIN = { style: BorderStyle.SINGLE, size: 8, color: "404040" };
function cellP(lines, opts) {
  opts = opts || {};
  if (typeof lines === "string") lines = [lines];
  return new TableCell({
    width: opts.w ? { size: opts.w, type: WidthType.PERCENTAGE } : undefined,
    verticalAlign: VerticalAlign.CENTER,
    margins: { top: 60, bottom: 60, left: 110, right: 110 },
    shading: opts.fill ? { type: ShadingType.CLEAR, fill: opts.fill } : undefined,
    borders: opts.borders,
    children: lines.map(function (ln) {
      return new Paragraph({
        alignment: opts.align || AlignmentType.LEFT,
        spacing: { line: 280, lineRule: "auto" },
        children: [new TextRun({ text: ln, bold: !!opts.bold, font: opts.bold ? FONT_HEAD : FONT_BODY, size: 21 })],
      });
    }),
  });
}
function threeLineTable(rows) {
  return new Table({
    width: { size: 100, type: WidthType.PERCENTAGE },
    borders: {
      top: { style: BorderStyle.SINGLE, size: 14, color: "000000" },
      bottom: { style: BorderStyle.SINGLE, size: 14, color: "000000" },
      left: NO_BORDER, right: NO_BORDER,
      insideHorizontal: NO_BORDER, insideVertical: NO_BORDER,
    },
    rows: rows.map(function (r, i) {
      return new TableRow({
        children: r.map(function (c) {
          var o = Object.assign({}, c);
          if (i === 0) {
            o.bold = true;
            o.align = AlignmentType.CENTER;
            o.borders = { top: NO_BORDER, bottom: THIN, left: NO_BORDER, right: NO_BORDER };
          }
          return cellP(o.lines, o);
        }),
      });
    }),
  });
}
function tableCaption(text) {
  return new Paragraph({
    alignment: AlignmentType.CENTER,
    spacing: { before: 160, after: 120 },
    children: [new TextRun({ text: text, font: FONT_HEAD, size: 21, bold: true })],
  });
}

module.exports = {
  FONT_BODY: FONT_BODY, FONT_HEAD: FONT_HEAD, FONT_CODE: FONT_CODE,
  readPng: readPng, figure: figure, caption: caption, codeCaption: codeCaption,
  t: t, b: b, P: P, H1: H1, H2: H2, H3: H3,
  codeBlock: codeBlock, refP: refP, spacer: spacer,
  coverLine: coverLine, coverTitle: coverTitle,
  NO_BORDER: NO_BORDER, THIN: THIN, cellP: cellP,
  threeLineTable: threeLineTable, tableCaption: tableCaption,
};
