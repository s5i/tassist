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

        const checksGroup = document.createElement('div');
        addEl.appendChild(checksGroup);
        checksGroup.classList.add('timer-checks-group');

        const loopLabel = document.createElement('label');
        checksGroup.appendChild(loopLabel);
        loopLabel.classList.add('timer-loop-label');
        const addLoopCb = document.createElement('input');
        addLoopCb.type = 'checkbox';
        loopLabel.appendChild(addLoopCb);
        loopLabel.appendChild(document.createTextNode('Loop'));

        const soundLabel = document.createElement('label');
        checksGroup.appendChild(soundLabel);
        soundLabel.classList.add('timer-loop-label');
        const addSoundCb = document.createElement('input');
        addSoundCb.type = 'checkbox';
        soundLabel.appendChild(addSoundCb);
        soundLabel.appendChild(document.createTextNode('Sound'));

        const addBtn = document.createElement('button');
        addEl.appendChild(addBtn);
        addBtn.textContent = 'Add';
        addBtn.classList.add('btn', 'timer-btn');
        addBtn.addEventListener('click', () => timers.hdlAdd(nameInput, hInput, mInput, sInput, addLoopCb, addSoundCb));

        timers.addRowEl = addEl;

        setInterval(timers.reload, 1000);
        timers.rebuildAll();
    },
    addRowEl: null,
    createRow: function (id) {
        const listEl = document.getElementById('timer-list');

        const entryEl = document.createElement('div');
        listEl.insertBefore(entryEl, timers.addRowEl);
        entryEl.classList.add('timer-entry');
        entryEl.dataset.id = id;
        entryEl.addEventListener('click', (ev) => {
            if (ev.target.closest('button, input, label')) return;
            timers.hdlAck(id);
        });

        const nameEl = document.createElement('span');
        entryEl.appendChild(nameEl);
        nameEl.classList.add('timer-name');

        const remainEl = document.createElement('span');
        entryEl.appendChild(remainEl);
        remainEl.classList.add('timer-remaining');

        const checksGroup = document.createElement('div');
        entryEl.appendChild(checksGroup);
        checksGroup.classList.add('timer-checks-group');

        const loopLabel = document.createElement('label');
        checksGroup.appendChild(loopLabel);
        loopLabel.classList.add('timer-loop-label');
        const loopCb = document.createElement('input');
        loopCb.type = 'checkbox';
        loopCb.addEventListener('change', () => timers.hdlLoop(id, loopCb.checked));
        loopLabel.appendChild(loopCb);
        loopLabel.appendChild(document.createTextNode('Loop'));

        const soundLabel = document.createElement('label');
        checksGroup.appendChild(soundLabel);
        soundLabel.classList.add('timer-loop-label');
        const soundCb = document.createElement('input');
        soundCb.type = 'checkbox';
        soundCb.addEventListener('change', () => timers.hdlSound(id, soundCb.checked));
        soundLabel.appendChild(soundCb);
        soundLabel.appendChild(document.createTextNode('Sound'));

        const startStopEl = document.createElement('button');
        entryEl.appendChild(startStopEl);
        startStopEl.classList.add('btn', 'timer-btn');

        const removeEl = document.createElement('button');
        entryEl.appendChild(removeEl);
        removeEl.textContent = 'Remove';
        removeEl.classList.add('btn', 'timer-btn');

        const refs = { entryEl, nameEl, remainEl, loopCb, soundCb, startStopEl, removeEl };
        timers.rows.set(id, refs);
        return refs;
    },
    updateRow: function (refs, t) {
        refs.nameEl.textContent = t.name;
        refs.remainEl.textContent = timers.fmtSec(t.active ? t.remaining : t.period);
        refs.remainEl.classList.toggle('timer-inactive', !t.active);
        refs.remainEl.classList.toggle('timer-firing', t.active && t.firing);
        refs.entryEl.classList.toggle('timer-entry-firing', t.active && t.firing);
        refs.loopCb.checked = t.loop;
        refs.soundCb.checked = t.sound;
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
    hdlAdd: async function (nameInput, hInput, mInput, sInput, loopCb, soundCb) {
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
        const loop = loopCb.checked;
        const sound = soundCb.checked;
        const r = await fetch('/api/timers/add', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name, period, loop, sound }) });
        if (r.ok) {
            toast.msg(`Timer "${name}" added.`);
            nameInput.value = '';
            hInput.value = '';
            mInput.value = '';
            sInput.value = '';
            loopCb.checked = false;
            soundCb.checked = false;
            timers.rebuildAll();
        } else {
            toast.msg('Error: ' + await r.text());
        }
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
timers.run();
