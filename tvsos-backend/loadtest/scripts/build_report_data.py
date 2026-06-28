#!/usr/bin/env python3
"""汇总 benchmark/k6 结果，并生成无需第三方库的答辩用 SVG 曲线。"""

import csv
import json
import math
import re
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
RESULTS = ROOT / "loadtest" / "results"
CHARTS = RESULTS / "charts"
CHARTS.mkdir(parents=True, exist_ok=True)


def build_algorithm_summary():
    pattern = re.compile(r"BenchmarkSchedulerVehicleScale/(NF|SA)/vehicles_(\d+)-\d+\s+\d+\s+(\d+) ns/op")
    values = defaultdict(list)
    source = RESULTS / "algorithm-benchmark.txt"
    if source.exists():
        for line in source.read_text(encoding="utf-8").splitlines():
            match = pattern.search(line)
            if match:
                values[(match.group(1), int(match.group(2)))].append(int(match.group(3)))

    rows = []
    for (algorithm, vehicles), samples in sorted(values.items()):
        avg_ns = sum(samples) / len(samples)
        rows.append({
            "algorithm": algorithm,
            "vehicles": vehicles,
            "avg_ms": avg_ns / 1_000_000,
            "orders_per_second": 5_000_000_000 / avg_ns,
            "samples": len(samples),
        })
    write_csv(RESULTS / "algorithm-summary.csv", rows,
              ["algorithm", "vehicles", "avg_ms", "orders_per_second", "samples"])
    return rows


def build_backlog_summary():
    points = defaultdict(list)
    for source in sorted((RESULTS / "raw").glob("backlog-rate*.csv")):
        rate_match = re.search(r"rate(\d+)", source.name)
        if not rate_match:
            continue
        rate = int(rate_match.group(1))
        with source.open(newline="", encoding="utf-8") as handle:
            reader = csv.DictReader(handle)
            for row in reader:
                if row.get("metric_name") != "pending_shipments":
                    continue
                timestamp = float(row.get("timestamp", 0))
                value = float(row.get("metric_value", 0))
                points[rate].append((timestamp, value))

    rows = []
    normalized = {}
    for rate, samples in sorted(points.items()):
        samples.sort()
        if not samples:
            continue
        start = samples[0][0]
        normalized[rate] = [(ts - start, value) for ts, value in samples]
        rows.append({
            "input_rate": rate,
            "start_pending": samples[0][1],
            "end_pending": samples[-1][1],
            "max_pending": max(value for _, value in samples),
            "growth": samples[-1][1] - samples[0][1],
        })
    write_csv(RESULTS / "backlog-summary.csv", rows,
              ["input_rate", "start_pending", "end_pending", "max_pending", "growth"])
    return normalized


def write_csv(path, rows, fields):
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)


def read_api_summary():
    path = RESULTS / "api-summary.csv"
    if not path.exists():
        return []
    with path.open(newline="", encoding="utf-8") as handle:
        return list(csv.DictReader(handle))


def svg_line_chart(path, title, x_label, y_label, series, log_y=False):
    width, height = 1000, 600
    left, right, top, bottom = 95, 35, 70, 80
    plot_w, plot_h = width - left - right, height - top - bottom
    colors = ["#2563eb", "#dc2626", "#16a34a", "#9333ea"]
    all_points = [point for points in series.values() for point in points]
    if not all_points:
        return
    xs = [p[0] for p in all_points]
    raw_ys = [max(p[1], 1e-9) for p in all_points]
    ys = [math.log10(y) for y in raw_ys] if log_y else raw_ys
    x_min, x_max = min(xs), max(xs)
    y_min, y_max = min(ys), max(ys)
    if x_min == x_max:
        x_max += 1
    if y_min == y_max:
        y_max += 1
    y_pad = (y_max - y_min) * 0.12
    y_min, y_max = y_min - y_pad, y_max + y_pad

    def sx(x): return left + (x - x_min) / (x_max - x_min) * plot_w
    def sy(y):
        value = math.log10(max(y, 1e-9)) if log_y else y
        return top + (y_max - value) / (y_max - y_min) * plot_h

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        f'<rect x="0" y="0" width="{width}" height="{height}" fill="#ffffff"/>',
        f'<text x="{width/2}" y="36" text-anchor="middle" font-size="25" font-family="sans-serif" font-weight="600">{title}</text>',
    ]
    for i in range(6):
        y = top + plot_h * i / 5
        value = y_max - (y_max - y_min) * i / 5
        label = f"{10**value:.3g}" if log_y else f"{value:.3g}"
        parts += [
            f'<line x1="{left}" y1="{y}" x2="{left+plot_w}" y2="{y}" stroke="#e5e7eb"/>',
            f'<text x="{left-12}" y="{y+5}" text-anchor="end" font-size="14" font-family="sans-serif">{label}</text>',
        ]
    parts += [
        f'<line x1="{left}" y1="{top}" x2="{left}" y2="{top+plot_h}" stroke="#374151"/>',
        f'<line x1="{left}" y1="{top+plot_h}" x2="{left+plot_w}" y2="{top+plot_h}" stroke="#374151"/>',
        f'<text x="{left+plot_w/2}" y="{height-24}" text-anchor="middle" font-size="17" font-family="sans-serif">{x_label}</text>',
        f'<text x="24" y="{top+plot_h/2}" transform="rotate(-90 24 {top+plot_h/2})" text-anchor="middle" font-size="17" font-family="sans-serif">{y_label}</text>',
    ]
    for i in range(6):
        value = x_min + (x_max - x_min) * i / 5
        x = sx(value)
        parts += [
            f'<line x1="{x}" y1="{top+plot_h}" x2="{x}" y2="{top+plot_h+7}" stroke="#374151"/>',
            f'<text x="{x}" y="{top+plot_h+26}" text-anchor="middle" font-size="14" font-family="sans-serif">{value:.3g}</text>',
        ]
    for index, (name, points) in enumerate(series.items()):
        color = colors[index % len(colors)]
        ordered = sorted(points)
        coords = " ".join(f"{sx(x):.1f},{sy(y):.1f}" for x, y in ordered)
        parts.append(f'<polyline points="{coords}" fill="none" stroke="{color}" stroke-width="3"/>')
        for x, y in ordered:
            parts.append(f'<circle cx="{sx(x):.1f}" cy="{sy(y):.1f}" r="5" fill="{color}"/>')
        lx = left + index * 180
        parts += [
            f'<line x1="{lx}" y1="55" x2="{lx+28}" y2="55" stroke="{color}" stroke-width="4"/>',
            f'<text x="{lx+36}" y="61" font-size="15" font-family="sans-serif">{name}</text>',
        ]
    parts.append('</svg>')
    path.write_text("\n".join(parts), encoding="utf-8")


def build_charts(algorithm_rows, backlog_points):
    api = read_api_summary()
    qps, p95 = defaultdict(list), defaultdict(list)
    for row in api:
        qps[row["endpoint"]].append((float(row["vus"]), float(row["qps"])))
        p95[row["endpoint"]].append((float(row["vus"]), float(row["p95_ms"])))
    svg_line_chart(CHARTS / "api-qps.svg", "并发用户数与 QPS", "并发用户数 (VU)", "QPS", qps)
    svg_line_chart(CHARTS / "api-p95.svg", "并发用户数与 P95 响应时间", "并发用户数 (VU)", "P95 (ms)", p95)

    algorithm = defaultdict(list)
    for row in algorithm_rows:
        algorithm[row["algorithm"]].append((row["vehicles"], row["avg_ms"]))
    svg_line_chart(CHARTS / "algorithm-time.svg", "车辆规模与调度计算时间", "车辆数量", "平均耗时 (ms，对数刻度)", algorithm, log_y=True)
    svg_line_chart(CHARTS / "backlog.svg", "持续发单下的待调度订单数量", "测试时间 (秒)", "待调度订单数",
                   {f"{rate} 单/秒": values for rate, values in backlog_points.items()})


if __name__ == "__main__":
    algorithm_rows = build_algorithm_summary()
    backlog_points = build_backlog_summary()
    build_charts(algorithm_rows, backlog_points)
    print(f"结果汇总完成：{RESULTS}")
