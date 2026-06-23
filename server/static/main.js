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

const containerHelp = {
    run: function () {
        document.querySelectorAll('.container[data-help]').forEach((container) => {
            const titleEl = container.querySelector('.container-title');
            const titleText = titleEl.textContent.trim();

            const titleSpan = document.createElement('span');
            titleSpan.className = 'container-title-text';
            titleSpan.textContent = titleText;
            titleEl.textContent = '';
            titleEl.appendChild(titleSpan);

            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'container-help-btn';
            btn.setAttribute('aria-label', 'Help');
            btn.textContent = '?';
            btn.addEventListener('click', (ev) => {
                ev.stopPropagation();
                containerHelp.show(container.dataset.help);
            });
            titleEl.appendChild(btn);
        });

        document.getElementById('help-prompt-close').addEventListener('click', containerHelp.hide);
        document.getElementById('help-prompt').addEventListener('click', (ev) => {
            if (ev.target.id === 'help-prompt') {
                containerHelp.hide();
            }
        });
    },
    show: function (helpId) {
        const section = document.getElementById('help-' + helpId);
        const container = document.querySelector(`.container[data-help="${helpId}"]`);
        if (!section || !container) return;

        const title = container.querySelector('.container-title-text').textContent;
        let html = section.innerHTML;

        document.getElementById('help-prompt-content').innerHTML = html;
        document.getElementById('help-prompt').classList.add('help-prompt-visible');
    },
    hide: function () {
        document.getElementById('help-prompt').classList.remove('help-prompt-visible');
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

const settingsNav = {
    run: function () {
        document.querySelectorAll('.settings-nav-item').forEach((btn) => {
            btn.addEventListener('click', settingsNav.hdlSwitch);
        });
    },
    hdlSwitch: function (ev) {
        const btn = ev.target.closest('.settings-nav-item');
        const panelId = btn.dataset.settings;

        document.querySelectorAll('.settings-nav-item').forEach((b) => b.classList.remove('settings-nav-active'));
        btn.classList.add('settings-nav-active');

        document.querySelectorAll('.settings-panel').forEach((p) => p.classList.remove('settings-panel-active'));
        document.getElementById('settings-panel-' + panelId).classList.add('settings-panel-active');
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

const loot = {
    items: {},
    prices: {},
    pasteEnabled: true,
    screenshots: [],
    currentIndex: -1,

    run: async function () {
        document.getElementById('loot-reset-prices').addEventListener('click', loot.resetPrices);
        document.getElementById('loot-prev-btn').addEventListener('click', () => loot.navigate(-1));
        document.getElementById('loot-next-btn').addEventListener('click', () => loot.navigate(1));
        document.getElementById('loot-upload-pick-btn').addEventListener('click', (ev) => {
            ev.stopPropagation();
            document.getElementById('loot-upload-file-input').click();
        });
        document.getElementById('loot-upload-area').addEventListener('click', () => {
            document.getElementById('loot-upload-file-input').click();
        });
        document.getElementById('loot-upload-file-input').addEventListener('change', loot.hdlFileInput);
        loot.bindUploadDragDrop();
        window.addEventListener('paste', loot.hdlPaste);
        window.addEventListener('keydown', loot.hdlKeydown);
        await loot.reload();
    },
    reload: async function () {
        const [itemsResp, pricesResp] = await Promise.all([
            fetch('/api/loot/items'),
            fetch('/api/loot/prices'),
        ]);
        if (!itemsResp.ok || !pricesResp.ok) {
            toast.msg('Failed to load loot data.');
            return;
        }
        const itemsList = await itemsResp.json();
        loot.items = Object.fromEntries(itemsList.map((item) => [String(item.id), item]));
        loot.prices = await pricesResp.json();
        loot.loadItemSettings();
        loot.refreshView();
    },
    getItemPrice: function (id) {
        const sid = String(id);
        if (sid in loot.prices) {
            return loot.prices[sid];
        }
        return loot.items[sid]?.value ?? 0;
    },
    getItemName: function (id) {
        return loot.items[String(id)]?.name ?? `Item ${id}`;
    },
    fmtGp: function (x) {
        return Number(x).toLocaleString();
    },
    calcCountsValue: function (counts) {
        let total = 0;
        Object.entries(counts).forEach(([itemId, count]) => {
            total += count * loot.getItemPrice(itemId);
        });
        return total;
    },
    calcItemCount: function (counts) {
        let total = 0;
        Object.values(counts).forEach((count) => {
            total += count;
        });
        return total;
    },
    addScreenshot: function (src, counts) {
        loot.screenshots.push({ src, counts });
        loot.currentIndex = loot.screenshots.length - 1;
        loot.refreshView();
    },
    navigate: function (delta) {
        if (loot.screenshots.length === 0) {
            return;
        }
        const next = loot.currentIndex + delta;
        if (next < 0 || next >= loot.screenshots.length) {
            return;
        }
        loot.currentIndex = next;
        loot.refreshDetails();
    },
    refreshView: function () {
        loot.refreshDetails();
        loot.refreshTotal();
    },
    refreshDetails: function () {
        const emptyEl = document.getElementById('loot-details-empty');
        const contentEl = document.getElementById('loot-details-content');
        const imageEl = document.getElementById('loot-current-image');
        const foundEl = document.getElementById('loot-current-found');
        const labelEl = document.getElementById('loot-screenshot-label');
        const prevBtn = document.getElementById('loot-prev-btn');
        const nextBtn = document.getElementById('loot-next-btn');

        const hasScreenshots = loot.screenshots.length > 0;
        if (hasScreenshots && loot.currentIndex < 0) {
            loot.currentIndex = loot.screenshots.length - 1;
        }
        emptyEl.toggleAttribute('hidden', hasScreenshots);
        contentEl.toggleAttribute('hidden', !hasScreenshots);
        prevBtn.disabled = !hasScreenshots || loot.currentIndex <= 0;
        nextBtn.disabled = !hasScreenshots || loot.currentIndex >= loot.screenshots.length - 1;

        if (!hasScreenshots) {
            labelEl.textContent = '—';
            return;
        }

        labelEl.textContent = `${loot.currentIndex + 1} / ${loot.screenshots.length}`;

        imageEl.innerHTML = '';
        foundEl.innerHTML = '';

        const shot = loot.screenshots[loot.currentIndex];
        const img = document.createElement('img');
        img.src = shot.src;
        imageEl.appendChild(img);

        let scrValue = 0;
        Object.entries(shot.counts).forEach(([itemId, count]) => {
            const name = loot.getItemName(itemId);
            const value = loot.getItemPrice(itemId);
            const ctVal = count * value;
            scrValue += ctVal;

            const itemLine = document.createElement('div');
            itemLine.className = 'loot-found-line';
            itemLine.textContent = `${name}: ${count} x ${loot.fmtGp(value)} gp = ${loot.fmtGp(ctVal)} gp`;
            foundEl.appendChild(itemLine);
        });

        const scrResult = document.createElement('div');
        scrResult.className = 'loot-found-line loot-found-bold';
        scrResult.textContent = `Value: ${loot.fmtGp(scrValue)} gp`;
        foundEl.appendChild(scrResult);

        const countLine = document.createElement('div');
        countLine.className = 'loot-found-line loot-found-bold';
        countLine.textContent = `Number of items detected: ${loot.calcItemCount(shot.counts)}`;
        foundEl.appendChild(countLine);
    },
    refreshTotal: function () {
        const emptyEl = document.getElementById('loot-total-empty');
        const foundEl = document.getElementById('loot-total-found');
        foundEl.innerHTML = '';

        if (loot.screenshots.length === 0) {
            emptyEl.toggleAttribute('hidden', false);
            foundEl.toggleAttribute('hidden', true);
            return;
        }

        emptyEl.toggleAttribute('hidden', true);
        foundEl.toggleAttribute('hidden', false);

        let totalValue = 0;
        loot.screenshots.forEach((shot, idx) => {
            const scrValue = loot.calcCountsValue(shot.counts);
            totalValue += scrValue;

            const line = document.createElement('div');
            line.className = 'loot-found-line';
            line.textContent = `Screenshot #${idx + 1}: ${loot.fmtGp(scrValue)} gp`;
            foundEl.appendChild(line);
        });

        const totalResult = document.createElement('div');
        totalResult.className = 'loot-found-line loot-found-bold';
        totalResult.textContent = `Total value: ${loot.fmtGp(totalValue)} gp`;
        foundEl.appendChild(totalResult);
    },
    loadItemSettings: function () {
        const categories = Object.fromEntries(
            Object.values(loot.items)
                .map((item) => item.category)
                .filter((v, idx, arr) => arr.indexOf(v) === idx)
                .sort()
                .map((category) => [
                    category,
                    Object.values(loot.items)
                        .filter((v) => v.category === category)
                        .sort((a, b) => a.name === b.name ? 0 : a.name < b.name ? -1 : 1)
                        .map((item) => String(item.id)),
                ])
        );

        const root = document.getElementById('loot-item-settings');
        root.innerHTML = '';

        const navDiv = document.createElement('div');
        navDiv.className = 'loot-category-nav';
        const contentDiv = document.createElement('div');
        contentDiv.className = 'loot-category-content';
        root.appendChild(navDiv);
        root.appendChild(contentDiv);

        const selectCategory = (navItem, cDiv) => {
            navDiv.querySelectorAll('.loot-category-nav-item').forEach((el) => el.classList.remove('active'));
            contentDiv.querySelectorAll('.loot-item-category').forEach((el) => el.classList.remove('active'));
            navItem.classList.add('active');
            cDiv.classList.add('active');
        };

        let firstNavItem = null;
        let firstCDiv = null;

        Object.entries(categories).forEach(([category, ids]) => {
            const cDiv = document.createElement('div');
            cDiv.className = 'loot-item-category';
            const navItem = document.createElement('div');
            navItem.className = 'loot-category-nav-item';
            navItem.textContent = category;
            navItem.addEventListener('click', () => selectCategory(navItem, cDiv));
            navDiv.appendChild(navItem);
            contentDiv.appendChild(cDiv);

            if (!firstNavItem) {
                firstNavItem = navItem;
                firstCDiv = cDiv;
            }

            ids.forEach((id) => {
                const item = loot.items[id];
                const row = document.createElement('div');
                row.className = 'loot-item-row';

                const name = document.createElement('span');
                name.className = 'loot-item-name';
                name.textContent = item.name;
                row.appendChild(name);

                const input = document.createElement('input');
                input.className = 'loot-item-value';
                input.id = `loot_${id}_value`;
                input.type = 'number';
                input.min = '0';
                input.value = loot.getItemPrice(id);
                if (item.market) {
                    input.classList.add('market-value');
                }
                if (String(id) in loot.prices) {
                    input.classList.add('modified-value');
                }
                input.addEventListener('change', () => loot.hdlPriceChange(id, input));
                row.appendChild(input);

                cDiv.appendChild(row);
            });
        });

        if (firstNavItem) {
            selectCategory(firstNavItem, firstCDiv);
        }
    },
    hdlPriceChange: async function (id, input) {
        const sid = String(id);
        const value = parseInt(input.value, 10) || 0;
        const defaultValue = loot.items[sid]?.value ?? 0;
        const toSave = (value > 0 && value !== defaultValue) ? value : 0;

        const r = await fetch('/api/loot/prices', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ prices: { [Number(sid)]: toSave } }),
        });
        if (!r.ok) {
            toast.msg('Error saving price: ' + await r.text());
            return;
        }
        loot.prices = await r.json();

        if (sid in loot.prices) {
            input.classList.add('modified-value');
        } else {
            input.classList.remove('modified-value');
        }
        loot.refreshView();
    },
    resetPrices: async function () {
        const r = await fetch('/api/loot/prices', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ reset: true }),
        });
        if (!r.ok) {
            toast.msg('Error: ' + await r.text());
            return;
        }
        loot.prices = await r.json();
        loot.loadItemSettings();
        loot.refreshView();
        toast.msg('Prices reset to defaults.');
    },
    bindUploadDragDrop: function () {
        const area = document.getElementById('loot-upload-area');
        area.addEventListener('dragover', (ev) => {
            ev.preventDefault();
            area.classList.add('drag-over');
        });
        area.addEventListener('dragleave', () => {
            area.classList.remove('drag-over');
        });
        area.addEventListener('drop', (ev) => {
            ev.preventDefault();
            area.classList.remove('drag-over');
            if (ev.dataTransfer.files.length > 0) {
                loot.processFile(ev.dataTransfer.files[0]);
            }
        });
    },
    hdlFileInput: function (ev) {
        const f = ev.target.files[0];
        ev.target.value = '';
        if (f) {
            loot.processFile(f);
        }
    },
    setUploading: function (uploading) {
        document.getElementById('loot-upload-area').classList.toggle('uploading', uploading);
        loot.pasteEnabled = !uploading;
    },
    processFile: async function (file) {
        if (!loot.pasteEnabled) {
            return;
        }
        if (file.type !== 'image/png') {
            toast.msg('Only PNG screenshots are supported.');
            return;
        }

        loot.setUploading(true);
        const start = performance.now();

        const formData = new FormData();
        formData.append('image', file);

        try {
            const r = await fetch('/api/loot/process', { method: 'POST', body: formData });
            if (!r.ok) {
                toast.msg('Error: ' + await r.text());
                return;
            }
            const counts = await r.json();
            loot.addScreenshot(URL.createObjectURL(file), counts);
            toast.msg(`Processing took ${((performance.now() - start) / 1000).toPrecision(2)}s.`);
        } catch (err) {
            toast.msg('Error: ' + err);
        } finally {
            loot.setUploading(false);
        }
    },
    hdlKeydown: function (event) {
        const lootTab = document.getElementById('tab-loot');
        if (!lootTab.classList.contains('tab-panel-active')) {
            return;
        }
        if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
            return;
        }
        if (event.key === 'ArrowLeft') {
            event.preventDefault();
            loot.navigate(-1);
        } else if (event.key === 'ArrowRight') {
            event.preventDefault();
            loot.navigate(1);
        }
    },
    hdlPaste: async function (event) {
        const lootTab = document.getElementById('tab-loot');
        if (!lootTab.classList.contains('tab-panel-active')) {
            return;
        }

        event.preventDefault();
        if (!loot.pasteEnabled) {
            return;
        }

        if (event.clipboardData.files.length === 0) {
            toast.msg('Non-screenshot paste detected.');
            return;
        }

        loot.processFile(event.clipboardData.files[0]);
    },
};

const hotkeys = {
    keyLabels: (function () {
        const labels = {};
        for (let i = 0; i < 12; i++) labels[i] = `F${i + 1}`;
        for (let i = 0; i < 12; i++) labels[i + 12] = `Shift+F${i + 1}`;
        for (let i = 0; i < 12; i++) labels[i + 24] = `Ctrl+F${i + 1}`;
        return labels;
    })(),
    getHk: function (hk, idx) {
        return hk[idx] || hk[String(idx)] || null;
    },
    run: function () {
        hotkeys.reloadList();
    },
    renderGrid: function (container, hk) {
        container.innerHTML = '';
        for (let idx = 0; idx < 36; idx++) {
            const row = document.createElement('div');
            row.className = 'hotkey-row';

            const label = document.createElement('span');
            label.className = 'hotkey-label';
            label.textContent = hotkeys.keyLabels[idx];
            row.appendChild(label);

            const entry = hotkeys.getHk(hk, idx);
            const val = document.createElement('span');
            val.className = 'hotkey-value';
            if (entry && entry.text) {
                val.textContent = entry.text;
            } else {
                val.classList.add('hotkey-value-empty');
            }
            row.appendChild(val);

            const asTag = document.createElement('span');
            asTag.className = 'hotkey-autosend-tag';
            if (entry && entry.text) {
                asTag.classList.add(entry.autosend ? 'hotkey-autosend-on' : 'hotkey-autosend-off');
                asTag.textContent = entry.autosend ? 'Send' : 'Type';
                asTag.title = entry.autosend ? 'Autosend enabled' : 'Autosend disabled (type only)';
            }
            row.appendChild(asTag);

            container.appendChild(row);
        }
    },
    renderEditGrid: function (container, hk) {
        container.innerHTML = '';
        for (let idx = 0; idx < 36; idx++) {
            const row = document.createElement('div');
            row.className = 'hotkey-row';

            const label = document.createElement('span');
            label.className = 'hotkey-label';
            label.textContent = hotkeys.keyLabels[idx];
            row.appendChild(label);

            const entry = hotkeys.getHk(hk, idx);
            const input = document.createElement('input');
            input.className = 'hotkey-input';
            input.type = 'text';
            input.value = (entry && entry.text) ? entry.text : '';
            input.dataset.idx = idx;
            row.appendChild(input);

            const asBtn = document.createElement('button');
            asBtn.type = 'button';
            asBtn.className = 'btn hotkey-autosend-btn';
            asBtn.dataset.idx = idx;
            const isOn = entry ? entry.autosend : true;
            hotkeys.setAutosendBtn(asBtn, isOn);
            asBtn.addEventListener('click', () => {
                const on = !asBtn.classList.contains('hotkey-autosend-btn-on');
                hotkeys.setAutosendBtn(asBtn, on);
            });
            row.appendChild(asBtn);

            container.appendChild(row);
        }
    },
    setAutosendBtn: function (btn, on) {
        btn.classList.toggle('hotkey-autosend-btn-on', on);
        btn.classList.toggle('hotkey-autosend-btn-off', !on);
        btn.textContent = on ? 'Send' : 'Type';
        btn.title = on ? 'Autosend enabled (click to toggle)' : 'Autosend disabled (click to toggle)';
    },
    collectEdits: function (gridEl) {
        const updated = {};
        gridEl.querySelectorAll('.hotkey-row').forEach((row) => {
            const input = row.querySelector('.hotkey-input');
            const asBtn = row.querySelector('.hotkey-autosend-btn');
            if (!input) return;
            const val = input.value.trim();
            if (val) {
                updated[Number(input.dataset.idx)] = {
                    text: val,
                    autosend: asBtn ? asBtn.classList.contains('hotkey-autosend-btn-on') : true,
                };
            }
        });
        return updated;
    },
    reloadList: async function () {
        const resp = await fetch('/api/hotkeys/list');
        if (!resp.ok) return;
        const d = await resp.json();

        const listEl = document.getElementById('hotkey-list');
        listEl.innerHTML = '';

        for (const entry of d) {
            const entryEl = document.createElement('div');
            entryEl.className = 'hotkey-entry';
            entryEl.dataset.id = entry.id;
            entryEl.dataset.name = entry.name;

            const headerEl = document.createElement('div');
            headerEl.className = 'hotkey-entry-header';
            entryEl.appendChild(headerEl);

            const nameEl = document.createElement('span');
            nameEl.className = 'hotkey-entry-name';
            nameEl.textContent = entry.name;
            headerEl.appendChild(nameEl);

            const btnGroup = document.createElement('div');
            btnGroup.className = 'hotkey-entry-btns';

            const loadBtn = document.createElement('button');
            loadBtn.textContent = 'Load';
            loadBtn.className = 'btn hotkey-btn';
            loadBtn.addEventListener('click', (ev) => { ev.stopPropagation(); hotkeys.hdlLoad(entry.id, entry.name); });
            btnGroup.appendChild(loadBtn);

            const editBtn = document.createElement('button');
            editBtn.textContent = 'Edit';
            editBtn.className = 'btn hotkey-btn';
            editBtn.addEventListener('click', (ev) => { ev.stopPropagation(); hotkeys.hdlEdit(entry.id); });
            btnGroup.appendChild(editBtn);

            const renameBtn = document.createElement('button');
            renameBtn.textContent = 'Rename';
            renameBtn.className = 'btn hotkey-btn';
            renameBtn.addEventListener('click', (ev) => { ev.stopPropagation(); hotkeys.hdlRename(entry.id, entry.name); });
            btnGroup.appendChild(renameBtn);

            const deleteBtn = document.createElement('button');
            deleteBtn.textContent = 'Delete';
            deleteBtn.className = 'btn hotkey-btn';
            deleteBtn.addEventListener('click', (ev) => { ev.stopPropagation(); hotkeys.hdlDelete(entry.id, entry.name); });
            btnGroup.appendChild(deleteBtn);

            headerEl.appendChild(btnGroup);

            headerEl.addEventListener('click', (ev) => {
                if (ev.target.closest('button')) return;
                hotkeys.hdlTogglePreview(entry.id);
            });

            const previewEl = document.createElement('div');
            previewEl.className = 'hotkey-preview';
            previewEl.hidden = true;
            entryEl.appendChild(previewEl);

            listEl.appendChild(entryEl);
        }

        {
            const storeEl = document.createElement('div');
            storeEl.className = 'hotkey-entry hotkey-entry-new';

            const nameInput = document.createElement('input');
            nameInput.className = 'hotkey-entry-name';
            nameInput.placeholder = 'Config name';
            storeEl.appendChild(nameInput);

            const storeBtn = document.createElement('button');
            storeBtn.textContent = 'Store';
            storeBtn.className = 'btn hotkey-btn';
            storeBtn.addEventListener('click', () => hotkeys.hdlStore(nameInput));
            storeEl.appendChild(storeBtn);

            listEl.appendChild(storeEl);
        }
    },
    hdlTogglePreview: async function (id) {
        const entryEl = document.querySelector(`.hotkey-entry[data-id="${id}"]`);
        if (!entryEl) return;
        if (entryEl.querySelector('.hotkey-edit-container')) return;
        const previewEl = entryEl.querySelector('.hotkey-preview');
        if (!previewEl) return;

        if (!previewEl.hidden) {
            previewEl.hidden = true;
            return;
        }

        if (!previewEl.dataset.loaded) {
            const r = await fetch('/api/hotkeys/detail', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
            if (!r.ok) {
                toast.msg('Error: ' + await r.text());
                return;
            }
            const cfg = await r.json();
            const gridEl = document.createElement('div');
            gridEl.className = 'hotkey-grid';
            hotkeys.renderGrid(gridEl, cfg.hotkeys);
            previewEl.appendChild(gridEl);
            previewEl.dataset.loaded = '1';
        }

        previewEl.hidden = false;
    },
    hdlStore: async function (nameInput) {
        const name = nameInput.value.trim() || 'Unnamed';
        const r = await fetch('/api/hotkeys/store', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) });
        if (r.ok) {
            toast.msg(`Stored "${name}".`);
            nameInput.value = '';
            hotkeys.reloadList();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlLoad: async function (id, name) {
        const r = await fetch('/api/hotkeys/load', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
        if (r.ok) {
            toast.msg(`Loaded "${name}".`);
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlRename: async function (id, name) {
        const newName = prompt(`Rename "${name}":`, name);
        if (newName === null) return;
        const trimmed = newName.trim();
        if (trimmed === '' || trimmed === name) return;

        const r = await fetch('/api/hotkeys/rename', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id, name: trimmed }) });
        if (r.ok) {
            toast.msg(`Renamed to "${trimmed}".`);
            hotkeys.reloadList();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlDelete: async function (id, name) {
        if (!confirm(`Delete "${name}"?`)) return;
        const r = await fetch('/api/hotkeys/delete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
        if (r.ok) {
            toast.msg(`Deleted "${name}".`);
            hotkeys.reloadList();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlEdit: async function (id) {
        const entryEl = document.querySelector(`.hotkey-entry[data-id="${id}"]`);
        if (!entryEl) return;

        let editContainer = entryEl.querySelector('.hotkey-edit-container');
        if (editContainer) {
            editContainer.remove();
            return;
        }

        const previewEl = entryEl.querySelector('.hotkey-preview');
        if (previewEl) previewEl.hidden = true;

        const r = await fetch('/api/hotkeys/detail', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
        if (!r.ok) {
            toast.msg('Error: ' + await r.text());
            return;
        }
        const cfg = await r.json();

        editContainer = document.createElement('div');
        editContainer.className = 'hotkey-edit-container';
        entryEl.appendChild(editContainer);

        const gridEl = document.createElement('div');
        gridEl.className = 'hotkey-grid';
        editContainer.appendChild(gridEl);
        hotkeys.renderEditGrid(gridEl, cfg.hotkeys);

        const saveBtn = document.createElement('button');
        saveBtn.className = 'btn hotkey-section-btn';
        saveBtn.textContent = 'Save changes';
        saveBtn.addEventListener('click', async () => {
            const updated = hotkeys.collectEdits(gridEl);
            const saveResp = await fetch('/api/hotkeys/update', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id, hotkeys: updated }) });
            if (saveResp.ok) {
                toast.msg('Config saved.');
                editContainer.remove();
                if (previewEl) { previewEl.dataset.loaded = ''; previewEl.innerHTML = ''; }
            } else {
                toast.msg('Error: ' + await saveResp.text());
            }
        });
        editContainer.appendChild(saveBtn);
    },
};

const clientPaths = {
    placeholders: {
        ancestra: 'e.g. C:\\Tibiantis\\',
        concordia: 'e.g. C:\\Tibiantis\\',
        relic: 'e.g. C:\\TibiaRelic\\',
    },
    run: async function () {
        const [pathsResp, presetsResp] = await Promise.all([
            fetch('/api/settings/client-paths'),
            fetch('/api/preset/list'),
        ]);
        if (!pathsResp.ok || !presetsResp.ok) return;
        const paths = await pathsResp.json();
        const presets = await presetsResp.json();

        const formEl = document.getElementById('client-paths-form');
        formEl.innerHTML = '';

        for (const id of presets.available) {
            const label = document.createElement('label');
            label.className = 'client-path-label';
            const span = document.createElement('span');
            span.textContent = id.charAt(0).toUpperCase() + id.slice(1);
            label.appendChild(span);

            const input = document.createElement('input');
            input.type = 'text';
            input.className = 'client-path-input';
            input.dataset.world = id;
            input.placeholder = clientPaths.placeholders[id] || 'Installation path';
            input.value = paths[id] || '';
            label.appendChild(input);

            formEl.appendChild(label);
        }

        const saveBtn = document.createElement('button');
        saveBtn.type = 'button';
        saveBtn.className = 'btn client-paths-save-btn';
        saveBtn.textContent = 'Save';
        saveBtn.addEventListener('click', clientPaths.hdlSave);
        formEl.appendChild(saveBtn);
    },
    hdlSave: async function () {
        const inputs = document.querySelectorAll('#client-paths-form .client-path-input');
        const payload = {};
        inputs.forEach((input) => {
            payload[input.dataset.world] = input.value.trim();
        });

        const r = await fetch('/api/settings/client-paths', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        if (r.ok) {
            toast.msg('Client paths saved.');
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
};

tabs.run();
containerHelp.run();
settingsNav.run();
preset.run();
updaterSettings.run();
clientPaths.run();
keepalive.run();
version.run();
update.run();
exp.run();
acc.run();
world.run();
timers.run();
loot.run();
hotkeys.run();
