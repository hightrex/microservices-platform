#!/bin/bash
set -euo pipefail

# -----------------------------------------------------------------------------
# MICROSERVICES PLATFORM — POSTGRESQL DATA DUMP (HTML + CSV)
#
# Outputs:
#  1) A single HTML report (easy to view)
#  2) Per-table CSV files (easy to filter/sort in spreadsheets)
#
# Defaults:
#  - Dumps auth_db + org_db (override via DB_INCLUDE or DUMP_ALL_USER_DBS=true)
#  - Limits HTML rows per table to ROW_LIMIT (default 500; set 0 for all)
#  - CSV defaults to ALL rows (set CSV_ROW_LIMIT to cap)
#
# Usage examples:
#   ./dump-postgres.sh
#   OUT_DIR=./artifacts ./dump-postgres.sh
#   ROW_LIMIT=0 ./dump-postgres.sh
#   CSV_ROW_LIMIT=10000 ./dump-postgres.sh
#   DB_INCLUDE='auth_db org_db' ./dump-postgres.sh
#   DUMP_ALL_USER_DBS=true ./dump-postgres.sh
# -----------------------------------------------------------------------------

# --- Configurable settings (override via env vars) ---
CONTAINER_NAME="${CONTAINER_NAME:-platform-postgres}"
DB_USER="${POSTGRES_USER:-postgres}"
SCHEMA="${PG_SCHEMA:-public}"

# Databases to include (space-separated). Default targets the services you care about.
DB_INCLUDE="${DB_INCLUDE:-auth_db org_db}"

# If true, ignore DB_INCLUDE and dump all non-template, non-system DBs.
DUMP_ALL_USER_DBS="${DUMP_ALL_USER_DBS:-false}"

# HTML safety: max rows per table in the HTML report. Set to 0 to dump all rows.
ROW_LIMIT="${ROW_LIMIT:-500}"

# CSV row limit (defaults to 0 = ALL rows). Set to a number to cap CSV sizes.
CSV_ROW_LIMIT="${CSV_ROW_LIMIT:-0}"

# Output location
OUT_DIR="${OUT_DIR:-.}"
TS="$(date +%Y%m%d-%H%M%S)"
OUT_FILE="${OUT_FILE:-$OUT_DIR/postgres_dump_${CONTAINER_NAME}_${TS}.html}"

# CSV output directory
CSV_DIR="${CSV_DIR:-$OUT_DIR/postgres_dump_${CONTAINER_NAME}_${TS}_csv}"

# --- Checks ---
require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: Required command '$1' not found in PATH."
    exit 1
  }
}
require_cmd podman

if ! podman container exists "$CONTAINER_NAME" >/dev/null 2>&1; then
  echo "ERROR: Podman container '$CONTAINER_NAME' not found."
  echo "Hint: 'podman ps --all' to confirm container name."
  exit 1
fi

if ! podman inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null | grep -q true; then
  echo "ERROR: Container '$CONTAINER_NAME' is not running."
  echo "Hint: start it first (e.g., 'make infra-up' or 'podman start $CONTAINER_NAME')."
  exit 1
fi

mkdir -p "$OUT_DIR"
mkdir -p "$CSV_DIR"

echo "Writing HTML report to: $OUT_FILE"
echo "Writing CSV files to:   $CSV_DIR"

# --- HTML header ---
cat > "$OUT_FILE" <<'HTML'
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>PostgreSQL Data Dump</title>
  <style>
    :root { color-scheme: light; }
    body {
      font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, "Apple Color Emoji","Segoe UI Emoji";
      margin: 20px;
      line-height: 1.35;
    }
    #top { position: absolute; top: 0; }
    h1 { margin: 0 0 6px 0; }
    .meta { color: #444; margin: 0 0 14px 0; }
    .toc {
      border: 1px solid #eee; border-radius: 10px; padding: 12px 14px; background: #fafafa;
      margin: 14px 0 24px 0;
    }
    .toc ul { margin: 8px 0 0 18px; }
    .toc code { font-size: 0.95em; }
    .db { margin-top: 26px; padding-top: 10px; border-top: 2px solid #ddd; }
    .table { margin: 14px 0 26px 0; }
    .table h3 { margin: 10px 0 6px 0; }
    .note { color: #666; font-size: 0.95em; margin: 6px 0 0 0; }
    .badge { display: inline-block; padding: 2px 8px; border-radius: 999px; background: #efefef; font-size: 12px; }
    .wrap { overflow-x: auto; border: 1px solid #eee; border-radius: 10px; padding: 8px; background: #fff; }
    table { border-collapse: collapse; font-size: 12px; }
    th, td { border: 1px solid #ccc; padding: 6px 8px; vertical-align: top; white-space: nowrap; }
    th { background: #f0f0f0; position: sticky; top: 0; z-index: 1; }
    details { margin: 10px 0; }
    summary { cursor: pointer; }
    .err {
      border: 1px solid #f5c2c7; background: #f8d7da; color: #842029;
      border-radius: 10px; padding: 10px 12px; margin-top: 8px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
      white-space: pre-wrap;
    }
    a { color: #0b5ed7; text-decoration: none; }
    a:hover { text-decoration: underline; }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; }
  </style>
</head>
<body>
<span id="top"></span>
HTML

# --- Report banner ---
{
  echo "<h1>MICROSERVICES PLATFORM — PostgreSQL Data Dump</h1>"
  echo "<p class=\"meta\"><b>Generated:</b> $(date) &nbsp; | &nbsp; <b>Container:</b> <code>${CONTAINER_NAME}</code> &nbsp; | &nbsp; <b>User:</b> <code>${DB_USER}</code> &nbsp; | &nbsp; <b>Schema:</b> <code>${SCHEMA}</code></p>"
  echo "<p class=\"meta\"><b>HTML row limit:</b> <code>${ROW_LIMIT}</code> &nbsp; | &nbsp; <b>CSV row limit:</b> <code>${CSV_ROW_LIMIT}</code> &nbsp; | &nbsp; <b>CSV dir:</b> <code>${CSV_DIR}</code></p>"
  echo "<p class=\"note\">Tips: Use <b>Ctrl+F</b> to search. Table headers stick while scrolling. CSV files contain all rows by default.</p>"
} >> "$OUT_FILE"

# --- Determine DB list ---
if [[ "$DUMP_ALL_USER_DBS" == "true" ]]; then
  DBS="$(
    podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -t -A -c \
      "SELECT datname
       FROM pg_database
       WHERE datistemplate = false
         AND datname NOT IN ('postgres','template0','template1')
       ORDER BY datname;"
  )"
else
  DBS="$(printf "%s\n" $DB_INCLUDE)"
fi

# --- Build TOC (database only) + validate DB existence ---
echo "<div class=\"toc\"><b>Contents</b><ul>" >> "$OUT_FILE"

valid_dbs=()
while IFS= read -r db; do
  [[ -z "$db" ]] && continue
  if ! podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -t -A -c "SELECT 1 FROM pg_database WHERE datname='${db}'" | grep -q 1; then
    echo "<li><code>${db}</code> — <span class=\"badge\">missing</span></li>" >> "$OUT_FILE"
    continue
  fi
  valid_dbs+=("$db")
  db_anchor="db-$(echo "$db" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9-_')"
  echo "<li><a href=\"#${db_anchor}\"><code>${db}</code></a></li>" >> "$OUT_FILE"
done <<< "$DBS"

echo "</ul></div>" >> "$OUT_FILE"

if [[ ${#valid_dbs[@]} -eq 0 ]]; then
  echo "<p><b>No valid databases found to dump.</b></p>" >> "$OUT_FILE"
  echo "</body></html>" >> "$OUT_FILE"
  echo "Done."
  exit 0
fi

# --- Helpers ---
sanitize_name() {
  # Lowercase + keep safe chars for filenames
  echo "$1" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9._-'
}

table_rowcount() {
  local db="$1" schema="$2" table="$3"
  podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$db" -t -A -c \
    "SELECT count(*)::bigint FROM \"${schema}\".\"${table}\";" 2>/dev/null || echo ""
}

dump_table_html() {
  local db="$1" schema="$2" table="$3" limit="$4"

  local query
  if [[ "$limit" == "0" ]]; then
    query="SELECT * FROM \"${schema}\".\"${table}\";"
  else
    query="SELECT * FROM \"${schema}\".\"${table}\" LIMIT ${limit};"
  fi

  set +e
  local out
  out="$(podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$db" -P footer=off -P border=1 -H -c "$query" 2>&1)"
  local rc=$?
  set -e

  if [[ $rc -ne 0 ]]; then
    echo "<div class=\"err\">ERROR querying ${schema}.${table}:\n${out}</div>"
    return 0
  fi

  echo "$out"
}

dump_table_csv() {
  local db="$1" schema="$2" table="$3" limit="$4" outfile="$5"

  local query
  if [[ "$limit" == "0" ]]; then
    query="COPY (SELECT * FROM \"${schema}\".\"${table}\") TO STDOUT WITH CSV HEADER;"
  else
    query="COPY (SELECT * FROM \"${schema}\".\"${table}\" LIMIT ${limit}) TO STDOUT WITH CSV HEADER;"
  fi

  set +e
  local out
  out="$(podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$db" -v ON_ERROR_STOP=1 -c "$query" 2>&1 >"$outfile")"
  local rc=$?
  set -e

  if [[ $rc -ne 0 ]]; then
    # If COPY failed, write the error text into a .error.txt file for visibility
    local errfile="${outfile}.error.txt"
    printf "%s\n" "$out" > "$errfile"
    # Ensure we don't leave a partial CSV that looks valid
    rm -f "$outfile"
    return 1
  fi

  return 0
}

# --- Dump each DB ---
for db in "${valid_dbs[@]}"; do
  db_anchor="db-$(echo "$db" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9-_')"

  echo "<div class=\"db\" id=\"${db_anchor}\">" >> "$OUT_FILE"
  echo "<h2>Database: <code>${db}</code></h2>" >> "$OUT_FILE"

  TABLES="$(
    podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$db" -t -A -c \
      "SELECT table_name
       FROM information_schema.tables
       WHERE table_schema = '${SCHEMA}'
         AND table_type = 'BASE TABLE'
       ORDER BY table_name;"
  )"

  if [[ -z "$TABLES" ]]; then
    echo "<p class=\"note\">No tables found in schema <code>${SCHEMA}</code>.</p></div>" >> "$OUT_FILE"
    continue
  fi

  # DB-specific CSV folder
  db_csv_dir="${CSV_DIR}/$(sanitize_name "$db")"
  mkdir -p "$db_csv_dir"

  echo "<p class=\"note\"><b>CSV output:</b> <code>${db_csv_dir}</code></p>" >> "$OUT_FILE"

  # Summary table (row counts + CSV links)
  echo "<details open><summary><b>Table summary</b> (row counts + CSV files)</summary>" >> "$OUT_FILE"
  echo "<div class=\"wrap\"><table><thead><tr><th>Table</th><th>Rows</th><th>CSV</th></tr></thead><tbody>" >> "$OUT_FILE"

  while IFS= read -r table; do
    [[ -z "$table" ]] && continue
    rcnt="$(table_rowcount "$db" "$SCHEMA" "$table")"
    [[ -z "$rcnt" ]] && rcnt="?"

    csv_name="$(sanitize_name "${SCHEMA}.${table}.csv")"
    csv_path="${db_csv_dir}/${csv_name}"

    # Generate CSV
    if dump_table_csv "$db" "$SCHEMA" "$table" "$CSV_ROW_LIMIT" "$csv_path"; then
      # Link is local path; browser will usually allow opening if file exists next to html.
      # We also show plain path for clarity.
      echo "<tr><td><code>${SCHEMA}.${table}</code></td><td>${rcnt}</td><td><code>${csv_path}</code></td></tr>" >> "$OUT_FILE"
    else
      echo "<tr><td><code>${SCHEMA}.${table}</code></td><td>${rcnt}</td><td><span class=\"badge\">CSV failed</span> <code>${csv_path}.error.txt</code></td></tr>" >> "$OUT_FILE"
    fi
  done <<< "$TABLES"

  echo "</tbody></table></div></details>" >> "$OUT_FILE"

  # Actual HTML table dumps
  while IFS= read -r table; do
    [[ -z "$table" ]] && continue

    rcnt="$(table_rowcount "$db" "$SCHEMA" "$table")"
    [[ -z "$rcnt" ]] && rcnt="?"

    echo "<div class=\"table\">" >> "$OUT_FILE"
    echo "<h3>Table: <code>${SCHEMA}.${table}</code> <span class=\"badge\">rows: ${rcnt}</span></h3>" >> "$OUT_FILE"

    if [[ "$ROW_LIMIT" != "0" ]]; then
      echo "<p class=\"note\">Showing up to <code>${ROW_LIMIT}</code> rows in HTML. Set <code>ROW_LIMIT=0</code> to dump everything (can be huge).</p>" >> "$OUT_FILE"
    fi

    echo "<div class=\"wrap\">" >> "$OUT_FILE"
    dump_table_html "$db" "$SCHEMA" "$table" "$ROW_LIMIT" >> "$OUT_FILE"
    echo "</div></div>" >> "$OUT_FILE"
  done <<< "$TABLES"

  echo "<p class=\"note\"><a href=\"#top\">Back to top</a></p>" >> "$OUT_FILE"
  echo "</div>" >> "$OUT_FILE"
done

echo "</body></html>" >> "$OUT_FILE"

echo "✅ Done."
echo "Open HTML report in a browser:"
echo "  $OUT_FILE"
echo ""
echo "CSV files written under:"
echo "  $CSV_DIR"
echo ""
echo "Useful overrides:"
echo "  ROW_LIMIT=0 ./dump-postgres.sh                   # dump all rows into HTML (can be huge)"
echo "  CSV_ROW_LIMIT=10000 ./dump-postgres.sh           # cap CSV rows (default is ALL)"
echo "  DB_INCLUDE='auth_db org_db' ./dump-postgres.sh   # only these DBs (default)"
echo "  DUMP_ALL_USER_DBS=true ./dump-postgres.sh        # dump all non-template DBs"
echo "  OUT_DIR=./artifacts ./dump-postgres.sh           # write under ./artifacts"
