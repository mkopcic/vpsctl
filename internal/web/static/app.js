// --- TAB MANAGEMENT ---

document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        const tab = btn.dataset.tab;

        document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));

        btn.classList.add('active');
        document.getElementById('tab-' + tab).classList.add('active');

        // Automatski ucitaj history kad se klikne taj tab
        if (tab === 'history') loadHistory();
    });
});

// Enter tipka u exec command polju pokrece Run
document.getElementById('exec-cmd').addEventListener('keydown', e => {
    if (e.key === 'Enter') runExec();
});

// --- STATE ---
let serverList = [];

// --- INIT ---
window.addEventListener('DOMContentLoaded', () => {
    loadServers();
});

// =============================================================
// SERVERS TAB
// =============================================================

async function loadServers() {
    const wrap = document.getElementById('servers-wrap');
    try {
        const res = await fetch('/servers');
        serverList = await res.json();
        renderServersTable(serverList, null);
        populateServerSelects();
    } catch (e) {
        wrap.innerHTML = `<p class="msg">Greska: ${e.message}</p>`;
    }
}

// Renderira tablicu servera, opcionally s ping rezultatima
function renderServersTable(servers, pingResults) {
    const wrap = document.getElementById('servers-wrap');

    // Napravi map: server name -> ping result za brzo trazenje
    const pingMap = {};
    if (pingResults) {
        pingResults.forEach(r => pingMap[r.server] = r);
    }

    const rows = servers.map(s => {
        const p = pingMap[s.name];
        let dotClass = 'dot-pending';
        let latencyCell = '<span style="color:var(--muted)">—</span>';

        if (p) {
            if (p.ok) {
                dotClass = 'dot-ok';
                const cls = p.latency_ms < 100 ? 'lat-fast' : p.latency_ms < 250 ? 'lat-mid' : 'lat-slow';
                latencyCell = `<span class="${cls}">${p.latency_ms}ms</span>`;
            } else {
                dotClass = 'dot-fail';
                latencyCell = '<span class="lat-slow">FAIL</span>';
            }
        }

        return `<tr>
            <td><span class="dot ${dotClass}"></span></td>
            <td>${s.name}</td>
            <td>${s.host}</td>
            <td>${s.port}</td>
            <td>${s.user}</td>
            <td>${latencyCell}</td>
        </tr>`;
    }).join('');

    wrap.innerHTML = `
        <table>
            <thead>
                <tr>
                    <th>Status</th>
                    <th>Name</th>
                    <th>Host</th>
                    <th>Port</th>
                    <th>User</th>
                    <th>Latency</th>
                </tr>
            </thead>
            <tbody>${rows}</tbody>
        </table>
    `;
}

// Ping All — GET /servers/ping, updatira tablicu s latencijama
async function pingAll() {
    const btn = document.getElementById('btn-ping');
    btn.disabled = true;
    btn.textContent = 'Pinging...';

    try {
        const res = await fetch('/servers/ping');
        const results = await res.json();
        renderServersTable(serverList, results);
    } catch (e) {
        alert('Ping failed: ' + e.message);
    } finally {
        btn.disabled = false;
        btn.textContent = 'Ping All';
    }
}

// =============================================================
// EXECUTE TAB
// =============================================================

// Popuni sve server dropdowne iz ucitane serverList
function populateServerSelects() {
    const options = serverList.map(s =>
        `<option value="${s.name}">${s.name}</option>`
    ).join('');

    document.getElementById('exec-server').innerHTML =
        '<option value="all">all (svi serveri)</option>' + options;

    document.getElementById('update-server').innerHTML =
        '<option value="all">all</option>' + options;

    document.getElementById('restart-server').innerHTML =
        '<option value="all">all</option>' + options;

    document.getElementById('history-server').innerHTML =
        '<option value="">all servers</option>' + options;
}

// POST /servers/exec
async function runExec() {
    const server  = document.getElementById('exec-server').value;
    const command = document.getElementById('exec-cmd').value.trim();
    const btn     = document.getElementById('btn-exec');
    const out     = document.getElementById('exec-output');

    if (!command) {
        document.getElementById('exec-cmd').focus();
        return;
    }

    btn.disabled = true;
    btn.textContent = 'Running...';

    showOutput(out, null); // clear + show loading

    try {
        const res = await fetch('/servers/exec', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ server, command })
        });
        const results = await res.json();
        showOutput(out, results);
    } catch (e) {
        out.innerHTML = `<span class="output-error">Error: ${e.message}</span>`;
    } finally {
        btn.disabled = false;
        btn.textContent = 'Run';
    }
}

// POST /servers/update
async function runUpdate() {
    const server = document.getElementById('update-server').value;
    const out    = document.getElementById('action-output');

    showOutput(out, null);

    try {
        const res = await fetch('/servers/update', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ server })
        });
        showOutput(out, await res.json());
    } catch (e) {
        out.innerHTML = `<span class="output-error">Error: ${e.message}</span>`;
    }
}

// POST /servers/restart
async function runRestart() {
    const server  = document.getElementById('restart-server').value;
    const service = document.getElementById('restart-service').value.trim();
    const out     = document.getElementById('action-output');

    if (!service) {
        document.getElementById('restart-service').focus();
        return;
    }

    showOutput(out, null);

    try {
        const res = await fetch('/servers/restart', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ server, service })
        });
        showOutput(out, await res.json());
    } catch (e) {
        out.innerHTML = `<span class="output-error">Error: ${e.message}</span>`;
    }
}

// Renderira SSH output u output box
// results = null -> loading state, results = array -> render
function showOutput(el, results) {
    el.classList.remove('hidden');

    if (!results) {
        el.innerHTML = '<span style="color:var(--muted)">Running...</span>';
        return;
    }

    const html = results.map(r => `
        <div class="output-block">
            <div class="output-server">${r.ok ? '&#x2713;' : '&#x2717;'} ${r.server}</div>
            ${r.ok
                ? `<div class="output-text">${escapeHtml(r.output)}</div>`
                : `<div class="output-error">${escapeHtml(r.error)}</div>`
            }
        </div>
    `).join('<hr>');

    el.innerHTML = html;
}

// =============================================================
// HISTORY TAB
// =============================================================

// GET /jobs?server=...&limit=...
async function loadHistory() {
    const wrap   = document.getElementById('history-wrap');
    const server = document.getElementById('history-server').value;
    const limit  = document.getElementById('history-limit').value || 20;

    wrap.innerHTML = '<p class="msg">Loading...</p>';

    try {
        let url = `/jobs?limit=${limit}`;
        if (server) url += `&server=${encodeURIComponent(server)}`;

        const res  = await fetch(url);
        const jobs = await res.json();

        if (!jobs || jobs.length === 0) {
            wrap.innerHTML = '<p class="msg">Nema jobova.</p>';
            return;
        }

        const rows = jobs.map(j => {
            const statusBadge = j.ok
                ? '<span class="badge badge-ok">OK</span>'
                : '<span class="badge badge-fail">FAIL</span>';

            const sourceBadge = j.source === 'api'
                ? '<span class="badge badge-api">api</span>'
                : '<span class="badge badge-cli">cli</span>';

            const cmd = j.command.length > 55
                ? j.command.substring(0, 52) + '...'
                : j.command;

            return `<tr>
                <td>${j.created_at}</td>
                <td>${statusBadge}</td>
                <td>${j.server}</td>
                <td>${j.duration_ms}ms</td>
                <td>${sourceBadge}</td>
                <td title="${escapeHtml(j.command)}">${escapeHtml(cmd)}</td>
            </tr>`;
        }).join('');

        wrap.innerHTML = `
            <table>
                <thead>
                    <tr>
                        <th>Time</th>
                        <th>Status</th>
                        <th>Server</th>
                        <th>Duration</th>
                        <th>Source</th>
                        <th>Command</th>
                    </tr>
                </thead>
                <tbody>${rows}</tbody>
            </table>
        `;
    } catch (e) {
        wrap.innerHTML = `<p class="msg">Error: ${e.message}</p>`;
    }
}

// =============================================================
// HELPERS
// =============================================================

// Escapira HTML specijalne znakove da sprijeci XSS
function escapeHtml(str) {
    if (!str) return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}
