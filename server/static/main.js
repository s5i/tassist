const listEl = document.getElementById('list');
const toastEl = document.getElementById('toast');
let toastTimer;

function toast(msg) {
    toastEl.textContent = msg;
    toastEl.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toastEl.classList.remove('show'), 2000);
}

async function hdlRenameStart(ev) {
    const entryEl = ev.target.closest('.entry');

    const nameEl = entryEl.querySelector('.name');
    nameEl.removeAttribute('readonly');
    nameEl.value = '';
    nameEl.focus({ focusVisible: true });
    nameEl.select();
}

async function hdlRenameDone(ev) {
    const entryEl = ev.target.closest('.entry');
    const id = entryEl.dataset.id;
    const name = entryEl.dataset.name;

    const nameEl = entryEl.querySelector('.name');
    const newName = nameEl.value.trim();

    if (newName == '' || newName == name) {
        nameEl.value = name;
        nameEl.setAttribute('readonly', true);
        return;
    }

    const r = await fetch('/api/rename', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id, name: newName }) });
    if (r.ok) {
        toast(`Renamed "${name}" to "${newName}".`);
        reload();
    } else {
        toast('Error: ' + await r.text());
    }
}

async function hdlLoad(ev) {
    const entryEl = ev.target.closest('.entry');
    const id = entryEl.dataset.id;
    const name = entryEl.dataset.name;

    const r = await fetch('/api/load', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
    if (r.ok) {
        toast(`Loaded "${name}".`);
    } else {
        toast('Error: ' + await r.text());
    };
}

async function hdlDelete(ev) {
    const entryEl = ev.target.closest('.entry');
    const id = entryEl.dataset.id;
    const name = entryEl.dataset.name;

    if (!confirm(`Delete "${name}"?`)) return;

    const r = await fetch('/api/delete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
    if (r.ok) {
        toast(`Deleted "${name}".`);
        reload();
    } else {
        toast('Error: ' + await r.text());
    }
}

async function reload() {
    const resp = await fetch('/api/list');
    const entries = await resp.json();
    listEl.innerHTML = '';

    for (const e of entries) {
        const id = e.id;
        const name = e.name;

        const entryEl = document.createElement('div');
        entryEl.className = 'entry';
        entryEl.setAttribute('data-id', id);
        entryEl.setAttribute('data-name', name);
        entryEl.innerHTML =
            '<input name="name" class="name" readonly="true" />' +
            '<button class="btn load">Load</button>' +
            '<button class="btn delete"">Delete</button>';

        const nameEl = entryEl.querySelector('.name');

        nameEl.value = name;
        nameEl.placeholder = name;
        nameEl.addEventListener('dblclick', hdlRenameStart);
        nameEl.addEventListener('focusout', hdlRenameDone);

        const loadEl = entryEl.querySelector('.load');
        loadEl.addEventListener('click', hdlLoad);

        const deleteEl = entryEl.querySelector('.delete');
        deleteEl.addEventListener('click', hdlDelete);

        listEl.appendChild(entryEl);
    }

    const entryEl = document.createElement('div');
    entryEl.className = 'entry';
    entryEl.innerHTML =
        '<input class="name" placeholder="New entry" />' +
        '<button class="btn store">Store</button>'

    const nameEl = entryEl.querySelector('.name');

    const storeEl = entryEl.querySelector('.store');
    storeEl.addEventListener('click', async (ev) => {
        console.log('storeEl.click', ev);

        const name = nameEl.value.trim() || 'Unnamed';
        const r = await fetch('/api/store', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name }) });
        if (r.ok) {
            toast(`Stored "${name}".`);
            reload();
        } else {
            toast('Error: ' + await r.text());
        }
    });

    listEl.appendChild(entryEl);
}

reload();
