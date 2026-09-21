package api

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

type dashboardLinkResponse struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Code         string    `json:"code"`
	ShortURL     string    `json:"short_url"`
	Destination  string    `json:"destination"`
	Active       bool      `json:"active"`
	Clicks       int       `json:"clicks"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	VersionCount int       `json:"version_count"`
}

type dashboardResponse struct {
	Links []dashboardLinkResponse `json:"links"`
}

type dashboardPageData struct {
	Links []dashboardLinkResponse
}

var dashboardPage = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>AtomicURL Dashboard</title>
  <style>
    :root { color-scheme: light dark; font: 16px/1.45 system-ui, sans-serif; }
    body { margin: 0; background: #f4f6f8; color: #18212b; }
    main { max-width: 1180px; margin: 0 auto; padding: 2rem 1rem 4rem; }
    .card { background: Canvas; border: 1px solid #d7dde3; border-radius: 12px; padding: 1rem; margin: 1rem 0; box-shadow: 0 2px 8px #0001; }
    h1, h2 { margin-top: 0; } h1 { display: inline-block; margin-right: 1rem; }
    label { display: block; font-weight: 600; margin: .5rem 0 .2rem; }
    input, button { box-sizing: border-box; font: inherit; padding: .55rem .7rem; border: 1px solid #aeb7c1; border-radius: 6px; }
    input { width: 100%; } button { cursor: pointer; background: #1769e0; color: white; border: 0; margin: .25rem .25rem .25rem 0; }
    button.secondary { background: #65717d; } button.danger { background: #b42318; }
    form.grid { display: grid; grid-template-columns: 1fr 2fr 1fr auto; gap: .5rem; align-items: end; }
    table { width: 100%; border-collapse: collapse; } th, td { text-align: left; vertical-align: top; padding: .7rem .5rem; border-bottom: 1px solid #d7dde3; }
    td.destination { max-width: 280px; overflow-wrap: anywhere; }
    .muted { color: #65717d; font-size: .9rem; } .inactive { opacity: .6; }
    .status { display: inline-block; border-radius: 99px; padding: .15rem .5rem; background: #d1fae5; color: #065f46; font-size: .85rem; }
    .status.off { background: #fee2e2; color: #991b1b; }
    #message { min-height: 1.5rem; font-weight: 600; } dialog { max-width: 700px; width: calc(100% - 2rem); }
    @media (max-width: 850px) { form.grid { grid-template-columns: 1fr; } table { display: block; overflow-x: auto; white-space: nowrap; } }
  </style>
</head>
<body>
<main>
  <h1>AtomicURL Dashboard</h1>
  <button class="secondary" onclick="location.reload()">Refresh</button>
  <p id="message" class="muted">Manage your permanent links.</p>

  <section class="card">
    <h2>Create a link</h2>
    <form id="create-form" class="grid">
      <div><label for="title">Title</label><input id="title" name="title" placeholder="My site"></div>
      <div><label for="destination">Destination</label><input id="destination" name="destination" type="url" required placeholder="https://example.com"></div>
      <div><label for="code">Custom code (optional)</label><input id="code" name="code" placeholder="my-site"></div>
      <button>Create link</button>
    </form>
  </section>

  <section class="card">
    <h2>Your links ({{len .Links}})</h2>
    {{if .Links}}
    <table>
      <thead><tr><th>Link</th><th>Destination</th><th>Status</th><th>Stats</th><th>Dates</th><th>Actions</th></tr></thead>
      <tbody>
      {{range .Links}}
      <tr class="{{if not .Active}}inactive{{end}}">
        <td><strong>{{if .Title}}{{.Title}}{{else}}Untitled{{end}}</strong><br><a href="/{{.Code}}" target="_blank" rel="noreferrer">/{{.Code}}</a><br><button class="secondary" onclick="copyText({{printf "%q" .ShortURL}})">Copy URL</button></td>
        <td class="destination"><a href="{{.Destination}}" target="_blank" rel="noreferrer">{{.Destination}}</a></td>
        <td><span class="status {{if not .Active}}off{{end}}">{{if .Active}}Active{{else}}Inactive{{end}}</span></td>
        <td>{{.Clicks}} clicks<br>{{.VersionCount}} versions<br><span class="muted">ID {{.ID}}</span></td>
        <td class="muted">Created {{.CreatedAt.Format "2006-01-02 15:04"}}<br>Updated {{.UpdatedAt.Format "2006-01-02 15:04"}}</td>
        <td><button onclick="editLink({{.ID}}, {{printf "%q" .Destination}})">Update destination</button><br><button class="secondary" onclick="showHistory({{.ID}})">View history</button></td>
      </tr>
      {{end}}
      </tbody>
    </table>
    {{else}}<p>No links yet. Create your first one above.</p>{{end}}
  </section>
</main>
<dialog id="edit-dialog"><form method="dialog" id="edit-form"><h2>Update destination</h2><input type="hidden" id="edit-id"><label for="edit-destination">Destination</label><input id="edit-destination" type="url" required><label for="edit-note">Note</label><input id="edit-note" required placeholder="Why are you changing it?"><p><button value="cancel" class="secondary">Cancel</button><button value="default">Save update</button></p></form></dialog>
<dialog id="history-dialog"><h2>Version history</h2><div id="history-content">Loading…</div><form method="dialog"><button class="secondary">Close</button></form></dialog>
<script>
const message = document.getElementById('message');
function setMessage(text, error = false) { message.textContent = text; message.style.color = error ? '#b42318' : ''; }
async function api(path, options = {}) {
  options.headers = Object.assign({'Accept': 'application/json'}, options.headers || {});
  const response = await fetch(path, options);
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error || 'Request failed');
  return body;
}
document.getElementById('create-form').addEventListener('submit', async (event) => {
  event.preventDefault(); const form = new FormData(event.target);
  try { await api('/links', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(Object.fromEntries(form))}); setMessage('Link created.'); location.reload(); }
  catch (error) { setMessage(error.message, true); }
});
function copyText(value) { navigator.clipboard?.writeText(value); setMessage('Short URL copied.'); }
function editLink(id, destination) { document.getElementById('edit-id').value = id; document.getElementById('edit-destination').value = destination; document.getElementById('edit-note').value = ''; document.getElementById('edit-dialog').showModal(); }
document.getElementById('edit-form').addEventListener('submit', async (event) => {
  event.preventDefault(); const id = document.getElementById('edit-id').value;
  try { await api('/links/' + id, {method: 'PATCH', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({destination: document.getElementById('edit-destination').value, note: document.getElementById('edit-note').value})}); setMessage('Link updated.'); location.reload(); }
  catch (error) { setMessage(error.message, true); }
});
async function showHistory(id) { const dialog = document.getElementById('history-dialog'); const content = document.getElementById('history-content'); dialog.showModal(); content.textContent = 'Loading…'; try { const data = await api('/links/' + id + '/history'); content.innerHTML = data.versions.length ? '<ol>' + data.versions.map(v => '<li><a href="' + escapeHTML(v.destination) + '" target="_blank" rel="noreferrer">' + escapeHTML(v.destination) + '</a>' + (v.active ? ' <strong>(current)</strong>' : '') + '<br><span class="muted">' + escapeHTML(v.note || 'No note') + ' · ' + new Date(v.created_at).toLocaleString() + '</span></li>').join('') + '</ol>' : '<p>No versions found.</p>'; } catch (error) { content.textContent = error.message; } }
function escapeHTML(value) { const div = document.createElement('div'); div.textContent = value; return div.innerHTML; }
</script>
</body>
</html>`))

func ShowUserDashBoard(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	var links []Link
	result := db.Preload("Versions").
		Where("user_id = ?", userID).
		Order("updated_at DESC, id DESC").
		Find(&links)
	if result.Error != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not load dashboard links.")
		return
	}

	response := dashboardResponse{Links: make([]dashboardLinkResponse, 0, len(links))}
	for _, link := range links {
		response.Links = append(response.Links, dashboardLinkResponse{
			ID:           link.ID,
			Title:        link.Title,
			Code:         link.Code,
			ShortURL:     dashboardShortURL(r, link.Code),
			Destination:  link.Destination,
			Active:       link.Active,
			Clicks:       link.Clicks,
			CreatedAt:    link.CreatedAt,
			UpdatedAt:    link.UpdatedAt,
			VersionCount: len(link.Versions),
		})
	}

	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := dashboardPage.Execute(w, dashboardPageData{Links: response.Links}); err != nil {
			return
		}
		return
	}

	writeJson(w, http.StatusOK, response)
}

func dashboardShortURL(r *http.Request, code string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:3000"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, host, code)
}
