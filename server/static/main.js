const toast = {
    msg: function (msg, timeout) {
        const toastEl = document.getElementById('toast');
        toastEl.innerHTML = msg;
        toastEl.classList.add('toast-visible');
        clearTimeout(toast.timer);
        if (!timeout) {
            timeout = 2000;
        }
        if (timeout > 0) {
            toast.timer = setTimeout(() => toastEl.classList.remove('toast-visible'), timeout);
        }
    },
    timer: undefined,
};

const keepalive = {
    run: async function () {
        // Foreground keepalive.
        setInterval(() => { fetch('/api/keepalive').catch(() => { window.close(); }); }, 1000);

        // Background keepalive; a little less intense.
        new Worker(window.URL.createObjectURL(new Blob([`setInterval(() => { fetch('${window.location.href}'+'api/keepalive').catch(); }, 10000);`], { type: "text/javascript" })));
    },
};

const version = {
    run: async function () {
        const resp = await fetch('/api/version');
        if (!resp.ok) return;
        const d = await resp.json()
        const version = d.version;

        document.getElementById('title').innerText = `TAssistant ${version}`;
    },
};

const update = {
    run: async function () {
        const resp = await fetch('/api/update/check');
        if (!resp.ok) return;
        const d = await resp.json()
        if (d.available) {
            toast.msg(`TAssistant ${d.version} available (see <a href="https://github.com/s5i/tassist/blob/main/CHANGELOG.md" target="_blank" class="link">changelog</a>). Click <a onclick="update.exec();" class="link">here</a> to update.`, 60000);
        }
    },
    exec: async function () {
        fetch('/api/update/execute').then(() => { window.close(); });
    }
};

const tabs = {
    run: function () {
        document.querySelectorAll('.tab-btn').forEach((btn) => {
            btn.addEventListener('click', tabs.hdlSwitch);
        });
    },
    hdlSwitch: function (ev) {
        const btn = ev.target.closest('.tab-btn');
        const tabId = btn.dataset.tab;

        document.querySelectorAll('.tab-btn').forEach((b) => b.classList.remove('tab-active'));
        btn.classList.add('tab-active');

        document.querySelectorAll('.tab-panel').forEach((p) => p.classList.remove('tab-panel-active'));
        document.getElementById('tab-' + tabId).classList.add('tab-panel-active');
    },
};

const exp = {
    run: function () {
        document.querySelectorAll('.exp-btn').forEach((x) => {
            x.addEventListener('click', exp.hdlGeneric);
        });

        setInterval(() => {
            if (Number(document.getElementById('exp-container').dataset.autorefresh)) {
                exp.reload();
            }
        }, 1000);

        exp.reload();
    },
    reload: async function () {
        const r = await fetch('/api/exp/stats');
        if (!r.ok) return;
        const d = await r.json();

        document.getElementById('exp-val-level').textContent = exp.fmtInt(d.level);
        document.getElementById('exp-val-total').textContent = exp.fmtInt(d.total_exp);
        document.getElementById('exp-val-remaining').textContent = exp.fmtInt(d.remaining_exp);
        document.getElementById('exp-val-session-delta').textContent = exp.fmtInt(d.session_delta);
        document.getElementById('exp-val-session-duration').textContent = exp.fmtDuration(d.session_duration_sec);
        document.getElementById('exp-val-session-rate').textContent = exp.fmtInt(d.session_rate);

        document.querySelectorAll('.exp-value').forEach((x) => {
            x.classList.remove('exp-value-paused');
            if (d.paused) {
                x.classList.add('exp-value-paused');
            }
        });

        document.getElementById('exp-btn-run').textContent = d.running ? 'Stop' : 'Start';
        document.getElementById('exp-btn-run').dataset.action = d.running ? '/api/exp/stop' : '/api/exp/start';
        document.getElementById('exp-btn-pause').textContent = d.paused ? 'Unpause' : 'Pause';
        document.getElementById('exp-btn-pause').dataset.action = d.paused ? '/api/exp/unpause' : '/api/exp/pause';
        document.getElementById('exp-btn-reset').dataset.action = '/api/exp/reset';
        document.getElementById('exp-container').dataset.autorefresh = Number(d.running && !d.paused);
    },
    fmtInt: function (x) {
        if (!Number.isInteger(x)) {
            return '-';
        }
        return x.toLocaleString('en-US');
    },
    fmtDuration: function (x) {
        if (!Number.isInteger(x)) {
            return '-';
        }
        const h = Math.floor(x / 3600);
        x -= h * 3600;
        const m = Math.floor(x / 60);
        x -= m * 60;
        const s = x;

        let durStr = '';
        if (h) durStr += `${h}h`;
        if (m) durStr += `${m}m`;
        if (s) durStr += `${s}s`;

        if (!durStr) return '-';
        return durStr;
    },
    hdlGeneric: async function (ev) {
        document.querySelectorAll('.exp-btn').forEach((x) => { x.disabled = true; });

        const btnEl = ev.target.closest('.exp-btn');
        fetch(btnEl.dataset.action).then(exp.reload);

        document.querySelectorAll('.exp-btn').forEach((x) => { x.removeAttribute('disabled'); });
    },
};

const acc = {
    run: function () {
        acc.reload();
    },
    reload: async function () {
        const resp = await fetch('/api/accounts/list');
        if (!resp.ok) return;
        const d = await resp.json();

        const accListEl = document.getElementById('acc-list');
        accListEl.innerHTML = '';

        for (const entry of d) {
            const id = entry.id;
            const name = entry.name;

            const entryEl = document.createElement('div');
            accListEl.appendChild(entryEl);
            entryEl.classList.add('acc-entry');
            entryEl.setAttribute('data-id', id);
            entryEl.setAttribute('data-name', name);

            const nameEl = document.createElement('input');
            entryEl.appendChild(nameEl);
            nameEl.classList.add('acc-name');
            nameEl.setAttribute('name', 'name');
            nameEl.setAttribute('readonly', true);
            nameEl.setAttribute('value', name);
            nameEl.setAttribute('placeholder', name);
            nameEl.addEventListener('dblclick', acc.hdlRenameStart);
            nameEl.addEventListener('focusout', acc.hdlRenameDone);

            const loadEl = document.createElement('button');
            entryEl.appendChild(loadEl);
            loadEl.textContent = 'Load';
            loadEl.classList.add('btn', 'acc-btn');
            loadEl.addEventListener('click', acc.hdlLoad);

            const deleteEl = document.createElement('button');
            entryEl.appendChild(deleteEl);
            deleteEl.textContent = 'Delete';
            deleteEl.classList.add('btn', 'acc-btn');
            deleteEl.addEventListener('click', acc.hdlDelete);
        }

        {
            const entryEl = document.createElement('div');
            accListEl.appendChild(entryEl);
            entryEl.classList.add('acc-entry');

            const nameEl = document.createElement('input');
            entryEl.appendChild(nameEl);
            nameEl.classList.add('acc-name');
            nameEl.setAttribute('placeholder', 'New entry');

            const storeEl = document.createElement('button');
            entryEl.appendChild(storeEl);
            storeEl.textContent = 'Store';
            storeEl.classList.add('btn', 'acc-btn');
            storeEl.addEventListener('click', acc.hdlStore);
        }
    },
    hdlRenameStart: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const nameEl = entryEl.querySelector('.acc-name');

        nameEl.removeAttribute('readonly');
        nameEl.value = '';
        nameEl.focus({ focusVisible: true });
        nameEl.select();
    },
    hdlRenameDone: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const id = entryEl.dataset.id;
        const name = entryEl.dataset.name;
        const nameEl = entryEl.querySelector('.acc-name');
        const newName = nameEl.value.trim();

        if (newName == '' || newName == name) {
            nameEl.value = name;
            nameEl.setAttribute('readonly', true);
            return;
        }

        const r = await fetch('/api/accounts/rename', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id, name: newName }) });
        if (r.ok) {
            toast.msg(`Renamed "${name}" to "${newName}".`);
            acc.reload();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlLoad: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const id = entryEl.dataset.id;
        const name = entryEl.dataset.name;

        const r = await fetch('/api/accounts/load', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
        if (r.ok) {
            toast.msg(`Loaded "${name}".`);
        } else {
            toast.msg('Error: ' + await r.text());
        };
    },
    hdlDelete: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const id = entryEl.dataset.id;
        const name = entryEl.dataset.name;

        if (!confirm(`Delete "${name}"?`)) return;

        const r = await fetch('/api/accounts/delete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
        if (r.ok) {
            toast.msg(`Deleted "${name}".`);
            acc.reload();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlStore: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const nameEl = entryEl.querySelector('.acc-name');
        const name = nameEl.value.trim() || 'Unnamed';

        const r = await fetch('/api/accounts/store', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name }) });
        if (r.ok) {
            toast.msg(`Stored "${name}".`);
            acc.reload();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
};

const world = {
    run: function () {
        setInterval(() => {
            world.reloadPing();
        }, 1000);

        setInterval(() => {
            world.reloadOnline();
        }, 60000);

        world.reloadPing();
        world.reloadOnline();

    },
    reloadPing: async function () {
        const r = await fetch('/api/world/ping');
        if (!r.ok) return;
        const d = await r.json();

        const rttEl = document.getElementById('world-val-rtt');
        const packetLossEl = document.getElementById('world-val-packetloss');

        const rtt = parseInt(d.rtt_msec);
        rttEl.textContent = world.fmtRTT(rtt, d.ok);
        rttEl.classList.remove('ping-value-ok', 'ping-value-meh', 'ping-value-bad');
        if (rtt > 200) {
            rttEl.classList.add('ping-value-bad');
        } else if (rtt > 100) {
            rttEl.classList.add('ping-value-meh');
        } else {
            rttEl.classList.add('ping-value-ok');
        }

        const packetLoss = parseFloat(d.packet_loss);
        packetLossEl.textContent = world.fmtPacketLoss(packetLoss, d.ok);
        packetLossEl.classList.remove('ping-value-ok', 'ping-value-meh', 'ping-value-bad');
        if (packetLoss > 0.1) {
            packetLossEl.classList.add('ping-value-bad');
        } else if (packetLoss > 0.05) {
            packetLossEl.classList.add('ping-value-meh');
        } else {
            packetLossEl.classList.add('ping-value-ok');
        }
    },
    reloadOnline: async function () {
        const r = await fetch('/api/world/online');
        if (!r.ok) return;
        const d = await r.json();

        const onlineEl = document.getElementById('world-val-online');

        onlineEl.textContent = world.fmtOnline(d.online, d.ok);
    },
    fmtOnline: function (x, ok) {
        if (!ok || !Number.isInteger(x)) {
            return '-';
        }
        return x;
    },
    fmtRTT: function (x, ok) {
        if (!ok || !Number.isInteger(x)) {
            return '-';
        }
        return `${x}ms`;
    },
    fmtPacketLoss: function (x, ok) {
        if (!ok || isNaN(x)) {
            return '-';
        }
        return `${(100.0 * x).toPrecision(1)}%`;
    },
};

const preset = {
    run: async function () {
        const resp = await fetch('/api/preset/list');
        if (!resp.ok) return;
        const d = await resp.json();

        const listEl = document.getElementById('preset-list');
        listEl.innerHTML = '';

        const indicatorEl = document.getElementById('preset-indicator');

        d.available.forEach((id) => {
            const btn = document.createElement('button');
            listEl.appendChild(btn);

            btn.textContent = preset.fmtID(id);
            btn.classList.add('btn', 'preset-btn');
            btn.dataset.id = id;
            btn.addEventListener('click', preset.hdlLoad);

            if (id == d.active) {
                btn.classList.add('preset-active');
                indicatorEl.textContent = preset.fmtID(id);
            }
        });
    },
    hdlLoad: async function (ev) {
        const btn = ev.target.closest('.preset-btn');
        const id = btn.dataset.id;

        const r = await fetch('/api/preset/switch', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
        if (r.ok) {
            toast.msg(`Switching to "${preset.fmtID(id)}" preset...`);
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    fmtID: function (id) {
        return String(id).charAt(0).toUpperCase() + String(id).slice(1);
    },
};

tabs.run();
preset.run();
keepalive.run();
version.run();
update.run();
exp.run();
acc.run();
world.run();
