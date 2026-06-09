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
    pendingVersion: undefined,
    run: function () {
        document.getElementById('update-prompt-update').addEventListener('click', update.exec);
        document.getElementById('update-prompt-skip').addEventListener('click', update.skip);
        document.getElementById('update-prompt-dismiss').addEventListener('click', update.hide);
        document.getElementById('update-prompt-disable').addEventListener('click', update.disableChecks);
        update.check();
    },
    check: async function () {
        const settingsResp = await fetch('/api/update/settings');
        if (!settingsResp.ok) return;
        const settings = await settingsResp.json();

        if (settings.mode !== 'Manual') return;

        const resp = await fetch('/api/update/check');
        if (!resp.ok) return;
        const d = await resp.json();
        if (d.available) {
            update.show(d.version);
        }
    },
    show: function (version) {
        update.pendingVersion = version;
        document.getElementById('update-prompt-title').textContent = `TAssistant ${version} available!`;
        document.getElementById('update-prompt').classList.add('update-prompt-visible');
    },
    hide: function () {
        document.getElementById('update-prompt').classList.remove('update-prompt-visible');
    },
    exec: async function () {
        update.hide();
        fetch('/api/update/execute').then(() => { window.close(); });
    },
    skip: async function () {
        const version = update.pendingVersion;
        if (!version) return;

        const r = await fetch('/api/update/skip', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ version }) });
        if (r.ok) {
            update.hide();
            toast.msg(`Skipped ${version}.`);
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    disableChecks: async function () {
        const r = await fetch('/api/update/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mode: 'Disabled' }),
        });
        if (r.ok) {
            update.hide();
            updaterSettings.reload();
            toast.msg('Update checks disabled. They can be re-enabled in the "Settings" tab.');
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
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

            const nameEl = document.createElement('span');
            entryEl.appendChild(nameEl);
            nameEl.classList.add('acc-name');
            nameEl.textContent = name;

            entryEl.addEventListener('click', (ev) => {
                if (ev.target.closest('button')) return;
                acc.hdlLoad(ev);
            });

            const renameEl = document.createElement('button');
            entryEl.appendChild(renameEl);
            renameEl.textContent = 'Rename';
            renameEl.classList.add('btn', 'acc-btn');
            renameEl.addEventListener('click', acc.hdlRename);

            const deleteEl = document.createElement('button');
            entryEl.appendChild(deleteEl);
            deleteEl.textContent = 'Delete';
            deleteEl.classList.add('btn', 'acc-btn');
            deleteEl.addEventListener('click', acc.hdlDelete);
        }

        {
            const entryEl = document.createElement('div');
            accListEl.appendChild(entryEl);
            entryEl.classList.add('acc-entry', 'acc-entry-new');

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
    hdlRename: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const id = entryEl.dataset.id;
        const name = entryEl.dataset.name;

        const newName = prompt(`Rename "${name}":`, name);
        if (newName === null) return;
        const trimmed = newName.trim();
        if (trimmed === '' || trimmed === name) return;

        const r = await fetch('/api/accounts/rename', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id, name: trimmed }) });
        if (r.ok) {
            toast.msg(`Renamed "${name}" to "${trimmed}".`);
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
        packetLossEl.textContent = world.fmtPacketLoss(packetLoss);
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
    fmtPacketLoss: function (x) {
        if (isNaN(x)) {
            return '-';
        }
        return `${(100.0 * x).toFixed(1)}%`;
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

const updaterSettings = {
    modes: ['Automatic', 'Manual', 'Disabled'],
    run: function () {
        const groupEl = document.getElementById('updater-mode-group');
        for (const mode of updaterSettings.modes) {
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.textContent = mode;
            btn.classList.add('btn', 'preset-btn');
            btn.dataset.mode = mode;
            btn.addEventListener('click', updaterSettings.hdlSelect);
            groupEl.appendChild(btn);
        }
        updaterSettings.reload();
    },
    reload: async function () {
        const resp = await fetch('/api/update/settings');
        if (!resp.ok) return;
        const d = await resp.json();
        document.querySelectorAll('#updater-mode-group .preset-btn').forEach((btn) => {
            btn.classList.toggle('preset-active', btn.dataset.mode === d.mode);
        });
    },
    hdlSelect: async function (ev) {
        const btn = ev.target.closest('.preset-btn');
        const mode = btn.dataset.mode;

        const r = await fetch('/api/update/settings', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mode }),
        });
        if (r.ok) {
            updaterSettings.reload();
        } else {
            toast.msg('Error: ' + await r.text());
            updaterSettings.reload();
        }
    },
};

const timers = {
    beepAudio: null,
    beepPlaying: false,
    rows: new Map(),
    run: function () {
        timers.beepAudio = new Audio('/beep.mp3');
        timers.beepAudio.loop = true;

        const listEl = document.getElementById('timer-list');
        const addEl = document.createElement('div');
        listEl.appendChild(addEl);
        addEl.classList.add('timer-add-entry');

        const nameInput = document.createElement('input');
        addEl.appendChild(nameInput);
        nameInput.classList.add('timer-add-name');
        nameInput.placeholder = 'Timer name';

        const toggleGroup = document.createElement('div');
        addEl.appendChild(toggleGroup);
        toggleGroup.classList.add('timer-toggle-group');

        const addLoopBtn = document.createElement('button');
        addLoopBtn.type = 'button';
        addLoopBtn.textContent = 'Loop';
        addLoopBtn.classList.add('btn', 'timer-btn', 'timer-toggle-btn');
        toggleGroup.appendChild(addLoopBtn);
        timers.setToggleBtn(addLoopBtn, false);
        addLoopBtn.addEventListener('click', () => timers.setToggleBtn(addLoopBtn, !addLoopBtn.classList.contains('timer-toggle-on')));

        const addSoundBtn = document.createElement('button');
        addSoundBtn.type = 'button';
        addSoundBtn.textContent = 'Sound';
        addSoundBtn.classList.add('btn', 'timer-btn', 'timer-toggle-btn');
        toggleGroup.appendChild(addSoundBtn);
        timers.setToggleBtn(addSoundBtn, false);
        addSoundBtn.addEventListener('click', () => timers.setToggleBtn(addSoundBtn, !addSoundBtn.classList.contains('timer-toggle-on')));

        const addAutoAckBtn = document.createElement('button');
        addAutoAckBtn.type = 'button';
        addAutoAckBtn.textContent = 'AutoAck';
        addAutoAckBtn.classList.add('btn', 'timer-btn', 'timer-toggle-btn');
        toggleGroup.appendChild(addAutoAckBtn);
        timers.setToggleBtn(addAutoAckBtn, false);
        addAutoAckBtn.addEventListener('click', () => timers.setToggleBtn(addAutoAckBtn, !addAutoAckBtn.classList.contains('timer-toggle-on')));

        const hmsBox = document.createElement('div');
        addEl.appendChild(hmsBox);
        hmsBox.classList.add('timer-add-hms');

        const hInput = document.createElement('input');
        hInput.type = 'number';
        hInput.min = '0';
        hInput.max = '99';
        hInput.placeholder = '00';
        hInput.classList.add('timer-add-hms-input');
        hmsBox.appendChild(hInput);

        const sep1 = document.createElement('span');
        sep1.textContent = ':';
        sep1.classList.add('timer-add-hms-sep');
        hmsBox.appendChild(sep1);

        const mInput = document.createElement('input');
        mInput.type = 'number';
        mInput.min = '0';
        mInput.max = '59';
        mInput.placeholder = '00';
        mInput.classList.add('timer-add-hms-input');
        hmsBox.appendChild(mInput);

        const sep2 = document.createElement('span');
        sep2.textContent = ':';
        sep2.classList.add('timer-add-hms-sep');
        hmsBox.appendChild(sep2);

        const sInput = document.createElement('input');
        sInput.type = 'number';
        sInput.min = '0';
        sInput.max = '59';
        sInput.placeholder = '00';
        sInput.classList.add('timer-add-hms-input');
        hmsBox.appendChild(sInput);

        [hInput, mInput, sInput].forEach((el, i, arr) => {
            el.addEventListener('input', () => {
                if (el.value.length >= 2 && i < arr.length - 1) {
                    arr[i + 1].focus();
                    arr[i + 1].select();
                }
            });
            el.addEventListener('click', () => {
                arr[i].select();
            });
        });

        const btnGroup = document.createElement('div');
        addEl.appendChild(btnGroup);
        btnGroup.classList.add('timer-btn-group');

        const addBtn = document.createElement('button');
        btnGroup.appendChild(addBtn);
        addBtn.textContent = 'Add';
        addBtn.classList.add('btn', 'timer-btn');
        addBtn.addEventListener('click', () => timers.hdlAdd(nameInput, hInput, mInput, sInput, addLoopBtn, addSoundBtn, addAutoAckBtn));

        timers.addRowEl = addEl;

        setInterval(timers.reload, 1000);
        timers.rebuildAll();
    },
    addRowEl: null,
    setToggleBtn: function (btn, on) {
        btn.classList.toggle('timer-toggle-on', on);
        btn.classList.toggle('timer-toggle-off', !on);
    },
    createRow: function (id) {
        const listEl = document.getElementById('timer-list');

        const entryEl = document.createElement('div');
        listEl.insertBefore(entryEl, timers.addRowEl);
        entryEl.classList.add('timer-entry');
        entryEl.dataset.id = id;
        entryEl.addEventListener('click', (ev) => {
            if (ev.target.closest('button, input')) return;
            timers.hdlAck(id);
        });

        const nameEl = document.createElement('span');
        entryEl.appendChild(nameEl);
        nameEl.classList.add('timer-name');

        const toggleGroup = document.createElement('div');
        entryEl.appendChild(toggleGroup);
        toggleGroup.classList.add('timer-toggle-group');

        const loopBtn = document.createElement('button');
        loopBtn.type = 'button';
        loopBtn.textContent = 'Loop';
        loopBtn.classList.add('btn', 'timer-btn', 'timer-toggle-btn');
        toggleGroup.appendChild(loopBtn);
        loopBtn.addEventListener('click', () => {
            const on = !loopBtn.classList.contains('timer-toggle-on');
            timers.setToggleBtn(loopBtn, on);
            timers.hdlLoop(id, on);
        });

        const soundBtn = document.createElement('button');
        soundBtn.type = 'button';
        soundBtn.textContent = 'Sound';
        soundBtn.classList.add('btn', 'timer-btn', 'timer-toggle-btn');
        toggleGroup.appendChild(soundBtn);
        soundBtn.addEventListener('click', () => {
            const on = !soundBtn.classList.contains('timer-toggle-on');
            timers.setToggleBtn(soundBtn, on);
            timers.hdlSound(id, on);
        });

        const autoAckBtn = document.createElement('button');
        autoAckBtn.type = 'button';
        autoAckBtn.textContent = 'AutoAck';
        autoAckBtn.classList.add('btn', 'timer-btn', 'timer-toggle-btn');
        toggleGroup.appendChild(autoAckBtn);
        autoAckBtn.addEventListener('click', () => {
            const on = !autoAckBtn.classList.contains('timer-toggle-on');
            timers.setToggleBtn(autoAckBtn, on);
            timers.hdlAutoAck(id, on);
        });

        const remainEl = document.createElement('span');
        entryEl.appendChild(remainEl);
        remainEl.classList.add('timer-remaining');

        const btnGroup = document.createElement('div');
        entryEl.appendChild(btnGroup);
        btnGroup.classList.add('timer-btn-group');

        const startStopEl = document.createElement('button');
        btnGroup.appendChild(startStopEl);
        startStopEl.classList.add('btn', 'timer-btn');

        const removeEl = document.createElement('button');
        btnGroup.appendChild(removeEl);
        removeEl.textContent = 'Remove';
        removeEl.classList.add('btn', 'timer-btn');

        const refs = { entryEl, nameEl, remainEl, loopBtn, soundBtn, autoAckBtn, startStopEl, removeEl };
        timers.rows.set(id, refs);
        return refs;
    },
    updateRow: function (refs, t) {
        refs.nameEl.textContent = t.name;
        refs.remainEl.textContent = timers.fmtSec(t.active ? t.remaining : t.period);
        refs.remainEl.classList.toggle('timer-inactive', !t.active);
        refs.remainEl.classList.toggle('timer-firing', t.active && t.firing);
        refs.entryEl.classList.toggle('timer-entry-firing', t.active && t.firing);
        timers.setToggleBtn(refs.loopBtn, t.loop);
        timers.setToggleBtn(refs.soundBtn, t.sound);
        timers.setToggleBtn(refs.autoAckBtn, t.auto_ack);
        refs.startStopEl.textContent = t.active ? 'Stop' : 'Start';
        refs.startStopEl.onclick = () => t.active ? timers.hdlStop(t.id) : timers.hdlStart(t.id);
        refs.removeEl.onclick = () => timers.hdlRemove(t.id, t.name);
    },
    rebuildAll: async function () {
        for (const [, refs] of timers.rows) refs.entryEl.remove();
        timers.rows.clear();

        const r = await fetch('/api/timers/list');
        if (!r.ok) return;
        const d = await r.json();

        for (const t of d) {
            const refs = timers.createRow(t.id);
            timers.updateRow(refs, t);
        }
        timers.updateBeep(d);
    },
    reload: async function () {
        const r = await fetch('/api/timers/list');
        if (!r.ok) return;
        const d = await r.json();

        for (const t of d) {
            const refs = timers.rows.get(t.id);
            if (refs) timers.updateRow(refs, t);
        }
        timers.updateBeep(d);
        timers.maybeAutoAck(d);
    },
    maybeAutoAck: function (d) {
        for (const t of d) {
            if (t.firing && t.auto_ack) {
                timers.hdlAck(t.id);
            }
        }
    },
    updateBeep: function (d) {
        const anyFiring = d.some(t => t.firing);
        const timersTabBtn = document.querySelector('.tab-btn[data-tab="timers"]');
        if (timersTabBtn) timersTabBtn.classList.toggle('tab-btn-firing', anyFiring);

        const shouldBeep = d.some(t => t.firing && t.sound);
        if (shouldBeep && !timers.beepPlaying) {
            timers.beepAudio.currentTime = 0;
            timers.beepAudio.play().catch(() => { });
            timers.beepPlaying = true;
        } else if (!shouldBeep && timers.beepPlaying) {
            timers.beepAudio.pause();
            timers.beepAudio.currentTime = 0;
            timers.beepPlaying = false;
        }
    },
    fmtSec: function (totalSec) {
        totalSec = Math.max(0, Math.floor(totalSec));
        const h = Math.floor(totalSec / 3600);
        const m = Math.floor((totalSec % 3600) / 60);
        const s = totalSec % 60;
        return String(h).padStart(2, '0') + ':' + String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0');
    },
    hdlStart: async function (id) {
        const r = await fetch('/api/timers/start', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
        if (!r.ok) toast.msg('Error: ' + await r.text());
    },
    hdlStop: async function (id) {
        const r = await fetch('/api/timers/stop', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
        if (!r.ok) toast.msg('Error: ' + await r.text());
    },
    hdlAck: async function (id) {
        const r = await fetch('/api/timers/ack', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
        if (!r.ok) toast.msg('Error: ' + await r.text());
    },
    hdlLoop: async function (id, loop) {
        const r = await fetch('/api/timers/loop', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id, loop }) });
        if (!r.ok) toast.msg('Error: ' + await r.text());
    },
    hdlSound: async function (id, sound) {
        const r = await fetch('/api/timers/sound', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id, sound }) });
        if (!r.ok) toast.msg('Error: ' + await r.text());
    },
    hdlAutoAck: async function (id, auto_ack) {
        const r = await fetch('/api/timers/autoack', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id, auto_ack }) });
        if (!r.ok) toast.msg('Error: ' + await r.text());
    },
    hdlRemove: async function (id, name) {
        if (!confirm(`Remove timer "${name}"?`)) return;
        const r = await fetch('/api/timers/remove', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
        if (r.ok) {
            toast.msg(`Timer "${name}" removed.`);
            timers.rebuildAll();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlAdd: async function (nameInput, hInput, mInput, sInput, loopBtn, soundBtn, autoAckBtn) {
        const name = nameInput.value.trim();
        const h = parseInt(hInput.value) || 0;
        const m = parseInt(mInput.value) || 0;
        const s = parseInt(sInput.value) || 0;
        const totalSec = h * 3600 + m * 60 + s;
        if (!name || totalSec <= 0) {
            toast.msg('Name and a positive duration are required.');
            return;
        }
        let period = '';
        if (h > 0) period += `${h}h`;
        if (m > 0) period += `${m}m`;
        if (s > 0 || period === '') period += `${s}s`;
        const loop = loopBtn.classList.contains('timer-toggle-on');
        const sound = soundBtn.classList.contains('timer-toggle-on');
        const auto_ack = autoAckBtn.classList.contains('timer-toggle-on');
        const r = await fetch('/api/timers/add', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name, period, loop, sound, auto_ack }) });
        if (r.ok) {
            toast.msg(`Timer "${name}" added.`);
            nameInput.value = '';
            hInput.value = '';
            mInput.value = '';
            sInput.value = '';
            timers.setToggleBtn(loopBtn, false);
            timers.setToggleBtn(soundBtn, false);
            timers.setToggleBtn(autoAckBtn, false);
            timers.rebuildAll();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
};

tabs.run();
preset.run();
updaterSettings.run();
keepalive.run();
version.run();
update.run();
exp.run();
acc.run();
world.run();
timers.run();
