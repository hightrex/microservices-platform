#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# -----------------------------------------------------------------------------
# MICROSERVICES PLATFORM — POSTGRESQL DATA DUMP (HTML + CSV)
#
# Outputs:
#   1) A single HTML report (easy to browse)
#   2) Per-table CSV files (easy to filter/sort in spreadsheets)
#
# Defaults:
#   - Tries DB_INCLUDE (default: auth_db org_db)
#   - If none of DB_INCLUDE exist, falls back to dumping ALL user DBs
#   - Limits HTML rows per table to ROW_LIMIT (default 500; set 0 for all)
#   - CSV defaults to ALL rows (set CSV_ROW_LIMIT to cap)
#
# Usage examples:
#   ./dump-db-data.sh
#   OUT_DIR=./artifacts ./dump-db-data.sh
#   ROW_LIMIT=0 ./dump-db-data.sh
#   CSV_ROW_LIMIT=10000 ./dump-db-data.sh
#   DB_INCLUDE='auth_db org_db' ./dump-db-data.sh
#   DUMP_ALL_USER_DBS=true ./dump-db-data.sh
#   TABLE_EXCLUDE_REGEX='^(schema_migrations)$' ./dump-db-data.sh
# -----------------------------------------------------------------------------

# --------------------------- Config (env overrides) ---------------------------
CONTAINER_NAME="${CONTAINER_NAME:-platform-postgres}"
DB_USER="${POSTGRES_USER:-postgres}"
SCHEMA="${PG_SCHEMA:-public}"

DB_INCLUDE="${DB_INCLUDE:-auth_db org_db}"
DUMP_ALL_USER_DBS="${DUMP_ALL_USER_DBS:-false}"

ROW_LIMIT="${ROW_LIMIT:-500}"          # HTML limit per table (0 = all)
CSV_ROW_LIMIT="${CSV_ROW_LIMIT:-0}"    # CSV limit per table (0 = all)

TABLE_INCLUDE_REGEX="${TABLE_INCLUDE_REGEX:-}"  # regex on table_name
TABLE_EXCLUDE_REGEX="${TABLE_EXCLUDE_REGEX:-}"  # regex on table_name

OUT_DIR="${OUT_DIR:-.}"
TS="$(date +%Y%m%d-%H%M%S)"
OUT_FILE="${OUT_FILE:-$OUT_DIR/postgres_dump_${CONTAINER_NAME}_${TS}.html}"
CSV_DIR="${CSV_DIR:-$OUT_DIR/postgres_dump_${CONTAINER_NAME}_${TS}_csv}"

# ------------------------------ Logging helpers ------------------------------
info() { printf 'ℹ️  %s\n' "$*"; }
ok()   { printf '✅ %s\n' "$*"; }
warn() { printf '⚠️  %s\n' "$*" >&2; }
die()  { printf '❌ %s\n' "$*" >&2; exit 1; }

require_cmd() { command -v "$1" >/dev/null 2>&1 || die "Required command '$1' not found in PATH."; }

# ------------------------------- Podman helpers ------------------------------
pod_psql() {
  # Usage: pod_psql <db_or_empty> <sql>
  local db="$1"; shift
  local sql="$1"
  if [[ -z "$db" ]]; then
    podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -t -A -c "$sql"
  else
    podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$db" -t -A -c "$sql"
  fi
}

sanitize_filename() { echo "$1" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9._-'; }
sanitize_anchor()   { echo "$1" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9-_'; }

list_user_dbs() {
  pod_psql "" \
    "SELECT datname
     FROM pg_database
     WHERE datistemplate = false
       AND datname NOT IN ('postgres','template0','template1')
     ORDER BY datname;" 2>/dev/null || true
}

db_exists() {
  local db="$1"
  pod_psql "" "SELECT 1 FROM pg_database WHERE datname='${db}';" 2>/dev/null | grep -q 1
}

schema_exists() {
  local db="$1" schema="$2"
  pod_psql "$db" "SELECT 1 FROM information_schema.schemata WHERE schema_name='${schema}';" | grep -q 1
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
  local out rc
  out="$(podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$db" -P footer=off -P border=1 -H -c "$query" 2>&1)"
  rc=$?
  set -e

  if [[ $rc -ne 0 ]]; then
    printf '<div class="errbox">ERROR querying %s.%s:\n%s</div>\n' "$schema" "$table" "$out"
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
  local err rc
  err="$(podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$db" -v ON_ERROR_STOP=1 -c "$query" \
    2>&1 >"$outfile")"
  rc=$?
  set -e

  if [[ $rc -ne 0 ]]; then
    printf "%s\n" "$err" > "${outfile}.error.txt"
    rm -f "$outfile"
    return 1
  fi
  return 0
}

# ------------------------------- Pre-flight ----------------------------------
require_cmd podman

podman container exists "$CONTAINER_NAME" >/dev/null 2>&1 \
  || die "Podman container '$CONTAINER_NAME' not found. Hint: 'podman ps --all'"

podman inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null | grep -q true \
  || die "Container '$CONTAINER_NAME' is not running. Hint: 'make infra-up' or 'podman start $CONTAINER_NAME'"

mkdir -p "$OUT_DIR" "$CSV_DIR"

info "Writing HTML report to: $OUT_FILE"
info "Writing CSV files to:   $CSV_DIR"
info "Filters:                include='${TABLE_INCLUDE_REGEX:-<none>}' exclude='${TABLE_EXCLUDE_REGEX:-<none>}'"

# ------------------------------- Decide DB list ------------------------------
DBS_RAW=""
if [[ "$DUMP_ALL_USER_DBS" == "true" ]]; then
  DBS_RAW="$(list_user_dbs)"
else
  DBS_RAW="$(printf '%s\n' $DB_INCLUDE)"
fi

# If DB_INCLUDE was used and none exist, auto-fallback to all user DBs
if [[ "$DUMP_ALL_USER_DBS" != "true" ]]; then
  any_ok=0
  while IFS= read -r db; do
    [[ -z "$db" ]] && continue
    if db_exists "$db"; then any_ok=1; break; fi
  done <<< "$DBS_RAW"

  if [[ "$any_ok" -eq 0 ]]; then
    warn "None of DB_INCLUDE databases exist: '${DB_INCLUDE}'. Falling back to DUMP_ALL_USER_DBS=true."
    DBS_RAW="$(list_user_dbs)"
  fi
fi

# ------------------------------- HTML header ---------------------------------
cat > "$OUT_FILE" <<'HTML'
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>PostgreSQL Data Dump</title>
  <style>
    :root{
      color-scheme: dark;
      --bg0:#070914; --bg1:#0b1020;
      --text: rgba(255,255,255,.92);
      --muted: rgba(255,255,255,.70);
      --border: rgba(255,255,255,.14);
      --border2: rgba(255,255,255,.10);
      --accent: #7aa2ff;
      --ok: #22c55e;
      --warn: #fbbf24;
      --danger: #fb7185;
      --shadow: 0 14px 40px rgba(0,0,0,.28);
      --radius: 14px;
      --mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
      --sans: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial;
    }

    body{
      font-family: var(--sans);
      margin: 22px;
      line-height: 1.45;
      color: var(--text);
      background:
        radial-gradient(1200px 700px at 20% 10%, rgba(122,162,255,.18), transparent 55%),
        radial-gradient(900px 700px at 60% 90%, rgba(251,113,133,.10), transparent 55%),
        linear-gradient(180deg, var(--bg0), var(--bg1));
      min-height: 100vh;
    }

    a{ color: var(--accent); text-decoration: none; }
    a:hover{ text-decoration: underline; }
    code{ font-family: var(--mono); font-size: .95em; }
    #top{ position: absolute; top: 0; }
    .container{ max-width: 1240px; margin: 0 auto; }

    .header{
      display:flex; gap:14px; align-items:flex-start; justify-content: space-between;
      padding: 16px 18px;
      border: 1px solid var(--border);
      border-radius: var(--radius);
      background: linear-gradient(180deg, rgba(255,255,255,.07), rgba(255,255,255,.03));
      box-shadow: var(--shadow);
      backdrop-filter: blur(10px);
    }
    .header h1{ margin:0; font-size: 20px; }
    .sub{ margin-top: 6px; color: var(--muted); font-size: 13px; }

    .chips{ display:flex; gap:8px; flex-wrap:wrap; justify-content:flex-end; }
    .chip{
      display:inline-flex; gap:8px; align-items:center;
      padding: 6px 10px;
      border-radius: 999px;
      background: rgba(255,255,255,.06);
      border: 1px solid var(--border2);
      color: var(--muted);
      font-size: 12px;
      white-space: nowrap;
    }
    .chip b{ color: var(--text); font-weight: 600; }

    .toolbar{
      margin-top: 12px;
      display:flex;
      gap:10px;
      align-items:center;
      justify-content: space-between;
      flex-wrap: wrap;
    }
    .search{
      flex: 1 1 520px;
      display:flex;
      align-items:center;
      gap:10px;
      padding: 10px 12px;
      border-radius: var(--radius);
      border: 1px solid var(--border);
      background: rgba(0,0,0,.22);
      box-shadow: 0 8px 22px rgba(0,0,0,.22);
      backdrop-filter: blur(10px);
    }
    .search input{
      width: 100%;
      border: 0;
      outline: 0;
      background: transparent;
      color: var(--text);
      font-size: 14px;
    }
    .search input::placeholder{ color: rgba(255,255,255,.45); }
    .search .hint{
      color: rgba(255,255,255,.55);
      font-size: 12px;
      white-space: nowrap;
    }
    .search kbd{
      font-family: var(--mono);
      font-size: 12px;
      padding: 1px 7px;
      border-radius: 7px;
      border: 1px solid var(--border2);
      background: rgba(255,255,255,.05);
      color: rgba(255,255,255,.82);
    }

    .toc{
      margin: 14px 0 22px 0;
      padding: 14px 16px;
      border-radius: var(--radius);
      border: 1px solid var(--border);
      background: rgba(255,255,255,.04);
      box-shadow: 0 10px 28px rgba(0,0,0,.22);
      backdrop-filter: blur(10px);
    }
    .toc ul{ margin: 10px 0 0 18px; }
    .toc li{ margin: 4px 0; }

    .db{
      margin-top: 24px;
      padding-top: 10px;
      border-top: 1px dashed rgba(255,255,255,.18);
    }

    .card{
      border: 1px solid var(--border);
      border-radius: var(--radius);
      background: rgba(255,255,255,.04);
      box-shadow: 0 10px 28px rgba(0,0,0,.18);
      padding: 12px 14px;
      margin: 12px 0;
      backdrop-filter: blur(10px);
    }

    details{
      margin: 12px 0;
      border: 1px solid var(--border2);
      border-radius: var(--radius);
      background: rgba(255,255,255,.03);
      overflow: hidden;
    }
    summary{
      cursor: pointer;
      padding: 10px 12px;
      list-style: none;
      user-select: none;
      color: var(--text);
      background: rgba(255,255,255,.04);
      border-bottom: 1px solid var(--border2);
      font-weight: 700;
      display:flex; align-items:center; justify-content: space-between;
      gap: 12px;
    }
    summary::-webkit-details-marker{ display:none; }
    .summary-meta{ color: var(--muted); font-weight: 500; font-size: 12px; }

    .note{ color: var(--muted); font-size: 13px; margin: 8px 0 0 0; }

    .badge{
      display:inline-block;
      padding: 2px 10px;
      border-radius: 999px;
      background: rgba(255,255,255,.07);
      border: 1px solid var(--border2);
      font-size: 12px;
      color: var(--muted);
    }
    .badge.ok{ color: rgba(34,197,94,.95); border-color: rgba(34,197,94,.25); background: rgba(34,197,94,.08); }
    .badge.warn{ color: rgba(251,191,36,.95); border-color: rgba(251,191,36,.25); background: rgba(251,191,36,.08); }
    .badge.err{ color: rgba(251,113,133,.95); border-color: rgba(251,113,133,.25); background: rgba(251,113,133,.08); }

    .wrap{
      overflow: auto;
      border-radius: calc(var(--radius) - 2px);
      border: 1px solid var(--border2);
      background: rgba(0,0,0,.20);
      margin: 10px 12px 12px 12px;
      max-height: 70vh;
    }

    table{
      border-collapse: separate;
      border-spacing: 0;
      width: max-content;
      min-width: 100%;
      font-size: 12px;
    }
    th, td{
      padding: 8px 10px;
      border-right: 1px solid rgba(255,255,255,.08);
      border-bottom: 1px solid rgba(255,255,255,.08);
      vertical-align: top;
      white-space: nowrap;
    }
    th{
      position: sticky;
      top: 0;
      z-index: 2;
      background: linear-gradient(180deg, rgba(255,255,255,.12), rgba(255,255,255,.06));
      color: rgba(255,255,255,.92);
      text-align: left;
      backdrop-filter: blur(8px);
    }
    tr:nth-child(even) td{ background: rgba(255,255,255,.02); }
    tr:hover td{ background: rgba(122,162,255,.07); }

    td:first-child, th:first-child{
      position: sticky;
      left: 0;
      z-index: 1;
      background: rgba(0,0,0,.36);
      backdrop-filter: blur(8px);
    }
    th:first-child{ z-index: 3; background: rgba(0,0,0,.48); }

    .errbox{
      border: 1px solid rgba(251,113,133,.35);
      background: rgba(251,113,133,.10);
      color: rgba(255,255,255,.92);
      border-radius: var(--radius);
      padding: 10px 12px;
      margin: 10px 12px 12px 12px;
      font-family: var(--mono);
      white-space: pre-wrap;
    }

    .footer-nav{
      margin: 6px 0 0 0;
      padding: 0 12px 12px 12px;
      color: var(--muted);
      font-size: 13px;
    }

    .k{
      display:inline-block;
      border: 1px solid var(--border2);
      background: rgba(255,255,255,.05);
      padding: 1px 7px;
      border-radius: 7px;
      font-family: var(--mono);
      font-size: 12px;
      color: rgba(255,255,255,.82);
    }

    .hidden{ display:none !important; }
    mark{
      background: rgba(122,162,255,.22);
      color: var(--text);
      padding: 0 2px;
      border-radius: 4px;
    }
  </style>
</head>
<body>
<span id="top"></span>
<div class="container">
HTML

# ----------------------------- Banner + Search -------------------------------
{
  echo "<div class=\"header\">"
  echo "  <div>"
  echo "    <h1>MICROSERVICES PLATFORM — PostgreSQL Data Dump</h1>"
  echo "    <div class=\"sub\">Generated: <span class=\"k\">$(date)</span> &nbsp; • &nbsp; Container: <span class=\"k\">${CONTAINER_NAME}</span> &nbsp; • &nbsp; User: <span class=\"k\">${DB_USER}</span> &nbsp; • &nbsp; Schema: <span class=\"k\">${SCHEMA}</span></div>"
  echo "    <div class=\"note\">Tips: <span class=\"k\">Ctrl+F</span> browser search • sticky headers/first column • CSV paths in summary</div>"
  echo "  </div>"
  echo "  <div class=\"chips\">"
  echo "    <span class=\"chip\"><b>HTML rows</b> <code>${ROW_LIMIT}</code></span>"
  echo "    <span class=\"chip\"><b>CSV rows</b> <code>${CSV_ROW_LIMIT}</code></span>"
  if [[ -n "$TABLE_INCLUDE_REGEX" ]]; then echo "    <span class=\"chip\"><b>include</b> <code>${TABLE_INCLUDE_REGEX}</code></span>"; fi
  if [[ -n "$TABLE_EXCLUDE_REGEX" ]]; then echo "    <span class=\"chip\"><b>exclude</b> <code>${TABLE_EXCLUDE_REGEX}</code></span>"; fi
  echo "    <span class=\"chip\"><b>CSV dir</b> <code>${CSV_DIR}</code></span>"
  echo "  </div>"
  echo "</div>"

  echo "<div class=\"toolbar\">"
  echo "  <div class=\"search\">"
  echo "    <span>🔎</span>"
  echo "    <input id=\"globalSearch\" type=\"search\" placeholder=\"Filter databases/tables… (e.g. users, org, sessions)\" autocomplete=\"off\" />"
  echo "    <span class=\"hint\"><kbd>/</kbd> focus • <kbd>Esc</kbd> clear</span>"
  echo "  </div>"
  echo "</div>"
} >> "$OUT_FILE"

# ----------------------------- TOC + validate DBs -----------------------------
echo "<div class=\"toc\" id=\"toc\"><b>Contents</b><ul id=\"tocList\">" >> "$OUT_FILE"

valid_dbs=()
while IFS= read -r db; do
  [[ -z "$db" ]] && continue
  if ! db_exists "$db"; then
    echo "<li data-filter=\"${db}\"><code>${db}</code> — <span class=\"badge err\">missing</span></li>" >> "$OUT_FILE"
    continue
  fi
  valid_dbs+=("$db")
  db_anchor="db-$(sanitize_anchor "$db")"
  echo "<li data-filter=\"${db}\"><a href=\"#${db_anchor}\"><code>${db}</code></a></li>" >> "$OUT_FILE"
done <<< "$DBS_RAW"

echo "</ul></div>" >> "$OUT_FILE"

if [[ ${#valid_dbs[@]} -eq 0 ]]; then
  avail="$(list_user_dbs)"
  {
    echo "<div class=\"card\"><b>No valid databases found to dump.</b>"
    echo "<div class=\"note\" style=\"margin-top:8px;\"><b>Requested:</b> <code>${DB_INCLUDE}</code></div>"
    echo "<div class=\"note\" style=\"margin-top:8px;\"><b>Available DBs in container:</b></div>"
    if [[ -n "$avail" ]]; then
      echo "<ul>"
      while IFS= read -r adb; do
        [[ -z "$adb" ]] && continue
        echo "<li><code>${adb}</code></li>"
      done <<< "$avail"
      echo "</ul>"
      echo "<div class=\"note\">Tip: run with <code>DUMP_ALL_USER_DBS=true</code> or set <code>DB_INCLUDE</code> to one of the DBs above.</div>"
    else
      echo "<div class=\"note\">Could not list DBs (psql query failed). Check credentials or container state.</div>"
    fi
    echo "</div></div></body></html>"
  } >> "$OUT_FILE"

  ok "Done (nothing to dump)."
  echo "Open HTML report:"
  echo "  $OUT_FILE"
  exit 0
fi

# ------------------------------- Dump each DB --------------------------------
for db in "${valid_dbs[@]}"; do
  db_anchor="db-$(sanitize_anchor "$db")"

  {
    echo "<div class=\"db\" id=\"${db_anchor}\" data-db=\"${db}\" data-filter=\"${db}\">"
    echo "<div class=\"card\">"
    echo "<h2 style=\"margin:0 0 6px 0; font-size:16px;\">Database: <code>${db}</code></h2>"
    echo "<div class=\"note\">CSV output folder: <code>${CSV_DIR}/$(sanitize_filename "$db")</code></div>"
    echo "</div>"
  } >> "$OUT_FILE"

  if ! schema_exists "$db" "$SCHEMA"; then
    {
      echo "<div class=\"card\"><div class=\"note\">Schema <code>${SCHEMA}</code> does not exist in this DB.</div></div>"
      echo "<div class=\"footer-nav\"><a href=\"#top\">Back to top</a></div>"
      echo "</div>"
    } >> "$OUT_FILE"
    continue
  fi

  where="table_schema = '${SCHEMA}' AND table_type = 'BASE TABLE'"
  if [[ -n "$TABLE_INCLUDE_REGEX" ]]; then where="${where} AND table_name ~ '${TABLE_INCLUDE_REGEX}'"; fi
  if [[ -n "$TABLE_EXCLUDE_REGEX" ]]; then where="${where} AND table_name !~ '${TABLE_EXCLUDE_REGEX}'"; fi

  TABLES="$(pod_psql "$db" \
    "SELECT table_name
     FROM information_schema.tables
     WHERE ${where}
     ORDER BY table_name;")"

  if [[ -z "$TABLES" ]]; then
    {
      echo "<div class=\"card\"><div class=\"note\">No tables found in schema <code>${SCHEMA}</code> (after filters).</div></div>"
      echo "<div class=\"footer-nav\"><a href=\"#top\">Back to top</a></div>"
      echo "</div>"
    } >> "$OUT_FILE"
    continue
  fi

  db_csv_dir="${CSV_DIR}/$(sanitize_filename "$db")"
  mkdir -p "$db_csv_dir"

  # Summary table (generated inline, no Python post-processing)
  {
    echo "<details open class=\"tbl-summary\" data-filter=\"${db}\">"
    echo "<summary><span>Table summary (row counts + CSV files)</span><span class=\"summary-meta\">${db}</span></summary>"
    echo "<div class=\"wrap\"><table><thead><tr><th>Table</th><th>Rows</th><th>CSV</th></tr></thead><tbody>"
  } >> "$OUT_FILE"

  while IFS= read -r table; do
    [[ -z "$table" ]] && continue

    rcnt="$(table_rowcount "$db" "$SCHEMA" "$table")"
    [[ -z "$rcnt" ]] && rcnt="?"

    csv_name="$(sanitize_filename "${SCHEMA}.${table}.csv")"
    csv_path="${db_csv_dir}/${csv_name}"

    # Generate CSV first (so the summary reflects reality)
    if dump_table_csv "$db" "$SCHEMA" "$table" "$CSV_ROW_LIMIT" "$csv_path"; then
      echo "<tr data-filter=\"${db} ${SCHEMA}.${table}\"><td><code>${SCHEMA}.${table}</code></td><td>${rcnt}</td><td><code>${csv_path}</code></td></tr>" >> "$OUT_FILE"
    else
      echo "<tr data-filter=\"${db} ${SCHEMA}.${table}\"><td><code>${SCHEMA}.${table}</code></td><td>${rcnt}</td><td><span class=\"badge err\">CSV failed</span> <code>${csv_path}.error.txt</code></td></tr>" >> "$OUT_FILE"
    fi
  done <<< "$TABLES"

  {
    echo "</tbody></table></div></details>"
  } >> "$OUT_FILE"

  # Actual HTML table dumps (this is the “simple HTML with many tables” part)
  while IFS= read -r table; do
    [[ -z "$table" ]] && continue

    t_anchor="${db_anchor}-t-$(sanitize_anchor "${SCHEMA}-${table}")"
    rcnt="$(table_rowcount "$db" "$SCHEMA" "$table")"
    [[ -z "$rcnt" ]] && rcnt="?"

    {
      echo "<div class=\"table\" id=\"${t_anchor}\" data-filter=\"${db} ${SCHEMA}.${table}\">"
      echo "<div class=\"card\" style=\"margin:12px 0;\">"
      echo "<div style=\"display:flex; gap:10px; align-items:center; justify-content:space-between; flex-wrap:wrap;\">"
      echo "  <div><b>Table:</b> <code>${SCHEMA}.${table}</code> &nbsp; <span class=\"badge\">rows: ${rcnt}</span></div>"
      if [[ "$ROW_LIMIT" != "0" ]]; then
        echo "  <div class=\"badge warn\">HTML shows up to ${ROW_LIMIT} rows</div>"
      else
        echo "  <div class=\"badge ok\">HTML shows all rows</div>"
      fi
      echo "</div>"
      echo "</div>"
      echo "<div class=\"wrap\">"
    } >> "$OUT_FILE"

    dump_table_html "$db" "$SCHEMA" "$table" "$ROW_LIMIT" >> "$OUT_FILE"

    {
      echo "</div>"
      echo "<div class=\"footer-nav\"><a href=\"#${db_anchor}\">Back to DB</a> &nbsp; • &nbsp; <a href=\"#top\">Back to top</a></div>"
      echo "</div>"
    } >> "$OUT_FILE"
  done <<< "$TABLES"

  {
    echo "<div class=\"footer-nav\"><a href=\"#top\">Back to top</a></div>"
    echo "</div>"
  } >> "$OUT_FILE"
done

# ------------------------------- JS: filter ----------------------------------
cat >> "$OUT_FILE" <<'HTML'
<script>
(() => {
  const $ = (sel, root=document) => root.querySelector(sel);
  const $$ = (sel, root=document) => Array.from(root.querySelectorAll(sel));
  const input = $("#globalSearch");
  if (!input) return;

  const tocItems = $$("#tocList li[data-filter]");
  const rows = $$("[data-filter]");
  const norm = (s) => (s || "").toLowerCase().trim();

  function applyFilter(v) {
    const q = norm(v);
    for (const el of rows) {
      const hay = norm(el.getAttribute("data-filter"));
      const ok = q === "" || hay.includes(q);
      el.classList.toggle("hidden", !ok);
    }
    for (const li of tocItems) {
      const hay = norm(li.getAttribute("data-filter"));
      const ok = q === "" || hay.includes(q);
      li.classList.toggle("hidden", !ok);
    }
  }

  document.addEventListener("keydown", (e) => {
    if (e.key === "/" && !(e.target && (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA"))) {
      e.preventDefault(); input.focus(); input.select();
    }
    if (e.key === "Escape" && document.activeElement === input) {
      input.value = ""; applyFilter(""); input.blur();
    }
  });

  input.addEventListener("input", () => applyFilter(input.value));
  applyFilter("");
})();
</script>
HTML

# ------------------------------- HTML footer ---------------------------------
{
  echo "<div class=\"card\" style=\"margin-top:18px;\">"
  echo "<b>Done.</b> <span class=\"note\">Open this report locally in a browser. CSV paths are shown in the summary for each DB.</span>"
  echo "</div>"
  echo "</div></body></html>"
} >> "$OUT_FILE"

ok "Done."
echo "Open HTML report:"
echo "  $OUT_FILE"
echo ""
echo "CSV files written under:"
echo "  $CSV_DIR"
