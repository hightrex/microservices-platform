#!/usr/bin/env python3
# scripts/security/kali/reports/generate-report.py

import json
import sys
import argparse
from pathlib import Path
from datetime import datetime

from jinja2 import Template
import yaml


class SecurityReportGenerator:
    """Aggregates findings from every scan module and generates HTML / JSON reports."""

    def __init__(self, results_dir: str):
        self.results_dir = Path(results_dir)
        self.findings: list[dict] = []

    # ------------------------------------------------------------------
    # Public
    # ------------------------------------------------------------------

    def parse_results(self) -> list[dict]:
        """Parse all result files and aggregate findings."""
        self._parse_nmap()
        self._parse_sqlmap()
        self._parse_nikto()
        self._parse_nuclei()
        self._parse_custom_tests()
        return self.findings

    # ------------------------------------------------------------------
    # Parsers
    # ------------------------------------------------------------------

    def _parse_nmap(self):
        """Parse nmap scan results (XML)."""
        nmap_file = self.results_dir / "recon" / "nmap-scan.xml"
        if not nmap_file.exists():
            return
        # Basic: look for open ports in the XML text
        content = nmap_file.read_text()
        # Count open ports (very rough heuristic)
        import re
        open_ports = re.findall(r'portid="(\d+)".*state="open"', content)
        if open_ports:
            self.findings.append({
                "title": f"Open ports detected: {', '.join(open_ports[:20])}",
                "severity": "INFO",
                "description": f"{len(open_ports)} open port(s) found by nmap",
                "remediation": "Review whether all open ports are necessary",
                "cwe": "CWE-200",
                "owasp": "Various",
            })

    def _parse_sqlmap(self):
        """Parse SQLMap results."""
        sqlmap_dir = self.results_dir / "injection" / "sqlmap"
        if not sqlmap_dir.exists():
            return

        for log_file in sqlmap_dir.glob("*/log"):
            content = log_file.read_text()
            if "sqlmap identified the following injection point" in content:
                self.findings.append({
                    "title": "SQL Injection Vulnerability",
                    "severity": "CRITICAL",
                    "description": f"SQL injection found in {log_file.parent.name}",
                    "evidence": content[:500],
                    "remediation": "Use parameterized queries",
                    "cwe": "CWE-89",
                    "owasp": "A03:2021 - Injection",
                })

    def _parse_nikto(self):
        """Parse Nikto scan results (JSON)."""
        nikto_file = self.results_dir / "web" / "nikto-report.json"
        if not nikto_file.exists():
            return

        try:
            data = json.loads(nikto_file.read_text())
            for vuln in data.get("vulnerabilities", []):
                severity = self._map_nikto_severity(vuln.get("OSVDB", ""))
                self.findings.append({
                    "title": vuln.get("msg", "Unknown"),
                    "severity": severity,
                    "description": vuln.get("msg", ""),
                    "url": vuln.get("url", ""),
                    "remediation": "Review Nikto documentation",
                    "owasp": "Various",
                })
        except (json.JSONDecodeError, KeyError) as exc:
            print(f"Warning: could not parse nikto report: {exc}", file=sys.stderr)

    def _parse_nuclei(self):
        """Parse Nuclei JSON-lines output."""
        nuclei_file = self.results_dir / "vuln" / "nuclei-report.json"
        if not nuclei_file.exists():
            return

        try:
            for line in nuclei_file.read_text().splitlines():
                if not line.strip():
                    continue
                entry = json.loads(line)
                severity = (entry.get("info", {}).get("severity", "info")).upper()
                self.findings.append({
                    "title": entry.get("info", {}).get("name", "Unknown"),
                    "severity": severity,
                    "description": entry.get("info", {}).get("description", ""),
                    "url": entry.get("matched-at", entry.get("host", "")),
                    "remediation": entry.get("info", {}).get("remediation", ""),
                    "cwe": entry.get("info", {}).get("classification", {}).get("cwe-id", [""])[0] if isinstance(entry.get("info", {}).get("classification", {}).get("cwe-id"), list) else "",
                    "owasp": "Various",
                })
        except (json.JSONDecodeError, KeyError) as exc:
            print(f"Warning: could not parse nuclei report: {exc}", file=sys.stderr)

    def _parse_custom_tests(self):
        """Parse findings.txt files written by custom test modules."""
        # Scan all module result dirs for findings.txt
        findings_files = list(self.results_dir.glob("*/findings.txt"))

        for findings_file in findings_files:
            module = findings_file.parent.name
            for line in findings_file.read_text().splitlines():
                line = line.strip()
                if not line:
                    continue

                parts = line.split(":", 1)
                if len(parts) == 2:
                    severity_label = parts[0].strip().upper()
                    message = parts[1].strip()
                else:
                    severity_label = "INFO"
                    message = line

                # Normalise severity
                severity = self._normalise_severity(severity_label)

                self.findings.append({
                    "title": f"[{module}] {message[:80]}",
                    "severity": severity,
                    "description": message,
                    "module": module,
                    "remediation": "Review and fix the identified issue",
                    "owasp": self._guess_owasp(module),
                })

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _map_nikto_severity(osvdb: str) -> str:
        """Map OSVDB ID to severity (heuristic)."""
        if not osvdb or osvdb == "0":
            return "LOW"
        return "MEDIUM"

    @staticmethod
    def _normalise_severity(label: str) -> str:
        """Map free-form severity strings to canonical values."""
        label = label.upper()
        for canonical in ("CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"):
            if canonical in label:
                return canonical
        if "POTENTIAL" in label:
            return "MEDIUM"
        return "INFO"

    @staticmethod
    def _guess_owasp(module: str) -> str:
        mapping = {
            "auth": "A01:2021 - Broken Access Control",
            "injection": "A03:2021 - Injection",
            "ssrf": "A10:2021 - SSRF",
            "api": "A01:2021 - Broken Access Control",
            "upload": "A04:2021 - Insecure Design",
        }
        return mapping.get(module, "Various")

    # ------------------------------------------------------------------
    # Report generators
    # ------------------------------------------------------------------

    def generate_html_report(self, output_file: str):
        """Generate HTML report."""
        template_str = """<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <title>Security Scan Report</title>
  <style>
    body { font-family: 'Segoe UI', sans-serif; margin: 24px; background: #f7f7fa; color: #222; }
    h1 { border-bottom: 2px solid #333; padding-bottom: 8px; }
    .summary { display: flex; gap: 16px; margin-bottom: 24px; }
    .card { padding: 12px 20px; border-radius: 6px; color: #fff; min-width: 100px; text-align: center; }
    .card.critical { background: #d32f2f; }
    .card.high     { background: #e65100; }
    .card.medium   { background: #f9a825; color: #333; }
    .card.low      { background: #1565c0; }
    .card.info     { background: #616161; }
    .card .num { font-size: 2em; font-weight: bold; }
    .finding { background: #fff; border-left: 4px solid #ccc; padding: 12px 16px; margin: 8px 0; border-radius: 4px; }
    .finding.CRITICAL { border-color: #d32f2f; }
    .finding.HIGH     { border-color: #e65100; }
    .finding.MEDIUM   { border-color: #f9a825; }
    .finding.LOW      { border-color: #1565c0; }
    .finding.INFO     { border-color: #616161; }
  </style>
</head>
<body>
  <h1>Security Scan Report</h1>
  <p>Date: {{ timestamp }}</p>
  <div class="summary">
    <div class="card critical"><div class="num">{{ summary.critical }}</div>Critical</div>
    <div class="card high"><div class="num">{{ summary.high }}</div>High</div>
    <div class="card medium"><div class="num">{{ summary.medium }}</div>Medium</div>
    <div class="card low"><div class="num">{{ summary.low }}</div>Low</div>
    <div class="card info"><div class="num">{{ summary.info }}</div>Info</div>
  </div>
  <h2>Findings ({{ summary.total }} total)</h2>
  {% for severity, items in findings_by_severity.items() %}
    {% if items %}
    <h3>{{ severity }} ({{ items|length }})</h3>
    {% for item in items %}
    <div class="finding {{ severity }}">
      <strong>{{ item.title }}</strong>
      <p>{{ item.description }}</p>
      {% if item.remediation %}<p><em>Remediation:</em> {{ item.remediation }}</p>{% endif %}
    </div>
    {% endfor %}
    {% endif %}
  {% endfor %}
</body>
</html>"""

        template_file = Path(__file__).parent / "templates" / "report.html.j2"
        if template_file.exists():
            template = Template(template_file.read_text())
        else:
            template = Template(template_str)

        by_severity = self._group_by_severity()
        summary = self._build_summary(by_severity)

        html = template.render(
            summary=summary,
            findings_by_severity=by_severity,
            timestamp=datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        )

        Path(output_file).write_text(html)

        # Also write a plain-text summary
        summary_file = Path(output_file).parent / "summary.txt"
        summary_file.write_text(
            f"Security Scan Summary\n"
            f"=====================\n"
            f"Date: {summary['scan_date']}\n"
            f"\n"
            f"Total Findings: {summary['total']}\n"
            f"  Critical: {summary['critical']}\n"
            f"  High:     {summary['high']}\n"
            f"  Medium:   {summary['medium']}\n"
            f"  Low:      {summary['low']}\n"
            f"  Info:     {summary['info']}\n"
        )

    def generate_json_report(self, output_file: str):
        """Generate JSON report."""
        by_severity = self._group_by_severity()
        report = {
            "scan_date": datetime.now().isoformat(),
            "findings": self.findings,
            "summary": {
                "total": len(self.findings),
                "by_severity": {
                    sev: len(items) for sev, items in by_severity.items()
                },
            },
        }
        Path(output_file).write_text(json.dumps(report, indent=2))

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _group_by_severity(self) -> dict[str, list]:
        groups: dict[str, list] = {
            "CRITICAL": [],
            "HIGH": [],
            "MEDIUM": [],
            "LOW": [],
            "INFO": [],
        }
        for finding in self.findings:
            sev = finding.get("severity", "INFO")
            groups.setdefault(sev, groups["INFO"]).append(finding)
        return groups

    def _build_summary(self, by_severity: dict[str, list]) -> dict:
        return {
            "total": len(self.findings),
            "critical": len(by_severity.get("CRITICAL", [])),
            "high": len(by_severity.get("HIGH", [])),
            "medium": len(by_severity.get("MEDIUM", [])),
            "low": len(by_severity.get("LOW", [])),
            "info": len(by_severity.get("INFO", [])),
            "scan_date": datetime.now().isoformat(),
        }


def main():
    parser = argparse.ArgumentParser(description="Generate security scan report")
    parser.add_argument("--input", required=True, help="Results directory")
    parser.add_argument("--output", required=True, help="Output file")
    parser.add_argument("--format", choices=["html", "json"], default="html")

    args = parser.parse_args()

    generator = SecurityReportGenerator(args.input)
    generator.parse_results()

    if args.format == "html":
        generator.generate_html_report(args.output)
    else:
        generator.generate_json_report(args.output)

    print(f"Report generated: {args.output}")


if __name__ == "__main__":
    main()
