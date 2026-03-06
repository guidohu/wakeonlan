document.addEventListener('DOMContentLoaded', () => {
    let currentHosts = [];
    const apiBase = '/api/hosts';
    const hostsContainer = document.getElementById('hosts-container');
    const addHostForm = document.getElementById('add-host-form');
    const toggleFormBtn = document.getElementById('toggle-form-btn');
    const closeFormBtn = document.getElementById('close-form-btn');
    const pingEnabledCheckbox = document.getElementById('host-ping-enabled');
    const modalOverlay = document.getElementById('modal-overlay');
    const toastContainer = document.getElementById('toast-container');
    const masterToggleCheckbox = document.getElementById('master-ping-enabled');

    let authToken = localStorage.getItem('wakeonlan_token');
    const loginModalOverlay = document.getElementById('login-modal-overlay');
    const loginForm = document.getElementById('login-form');

    const showLogin = () => {
        loginModalOverlay.classList.remove('hidden');
        document.getElementById('login-username').focus();
    };

    const hideLogin = () => {
        loginModalOverlay.classList.add('hidden');
    };

    const apiFetch = async (url, options = {}) => {
        if (!options.headers) options.headers = {};
        if (authToken) {
            options.headers['Authorization'] = `Bearer ${authToken}`;
        }
        const res = await fetch(url, options);
        if (res.status === 401 || res.status === 403) {
            authToken = null;
            localStorage.removeItem('wakeonlan_token');
            showLogin();
            throw new Error('Session expired or unauthorized');
        }
        return res;
    };

    // Form Toggle
    const toggleForm = () => {
        modalOverlay.classList.toggle('hidden');
        if (!modalOverlay.classList.contains('hidden')) {
            document.getElementById('host-name').focus();
        }
    };

    toggleFormBtn.addEventListener('click', toggleForm);
    closeFormBtn.addEventListener('click', () => {
        modalOverlay.classList.add('hidden');
        addHostForm.reset();
        pingEnabledCheckbox.checked = true;
        delete addHostForm.dataset.editingId;
        document.querySelector('#modal-title').textContent = 'Add New Host';
        addHostForm.querySelector('button[type="submit"]').textContent = 'Save Host';
    });

    // Close modal on outside click
    modalOverlay.addEventListener('click', (e) => {
        if (e.target === modalOverlay) {
            closeFormBtn.click();
        }
    });

    // Close modal on Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && !modalOverlay.classList.contains('hidden')) {
            closeFormBtn.click();
        }
    });

    // Toasts
    const showToast = (message, type = 'success') => {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;

        const icon = type === 'success' ? '✓' : '⚠️';
        toast.innerHTML = `<span>${icon}</span> <span>${message}</span>`;

        toastContainer.appendChild(toast);

        setTimeout(() => {
            toast.classList.add('fade-out');
            toast.addEventListener('animationend', () => toast.remove());
        }, 3000);
    };

    // Render Host Card
    const getSafeURL = (urlStr) => {
        if (!urlStr) return '#';
        try {
            const parsed = new URL(urlStr);
            if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
                return urlStr;
            }
        } catch (e) {
            // Invalid URL
        }
        return '#';
    };

    const createHostCard = (host) => {
        const card = document.createElement('div');
        card.className = 'host-card glass-panel';
        card.dataset.id = host.id;

        const safeUrl = getSafeURL(host.access_url);

        const titleHtml = host.access_url
            ? `<h3><a href="${escapeHTML(safeUrl)}" target="_blank" rel="noopener noreferrer" style="color: inherit; text-decoration: none; display: inline-flex; align-items: center; gap: 6px;">${escapeHTML(host.name)} <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="opacity: 0.5;"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path><polyline points="15 3 21 3 21 9"></polyline><line x1="10" y1="14" x2="21" y2="3"></line></svg></a></h3>`
            : `<h3>${escapeHTML(host.name)}</h3>`;

        card.innerHTML = `
            <div class="host-info">
                ${titleHtml}
                <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                    <div>
                        <p title="MAC Address">${escapeHTML(host.mac_address)}</p>
                        ${host.ip ? `<p title="Host IP">${escapeHTML(host.ip)}</p>` : ''}
                    </div>
                    ${host.ip ? `
                    <div style="display: flex; flex-direction: column; align-items: center; gap: 6px;" title="Toggle Monitoring">
                        <span style="font-size: 0.75rem; color: var(--text-secondary); font-weight: 500;">Monitoring</span>
                        <label class="switch">
                            <input type="checkbox" onchange="toggleMonitoring('${host.id}', this)" ${host.ping_enabled ? 'checked' : ''}>
                            <span class="slider round success-toggle"></span>
                        </label>
                    </div>` : ''}
                </div>
            </div>
            <div class="ping-container" id="ping-container-${host.id}" style="${!host.ping_enabled ? 'opacity: 0.3;' : ''}">
                ${host.ip ? `<div class="ping-track"></div>` : ''}
            </div>
            <div class="host-actions">
                <button class="btn wake-btn" onclick="wakeHost('${host.id}', this)" title="Send Magic Packet">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="margin-right: 6px; vertical-align: text-bottom;">
                        <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
                    </svg> Wake
                </button>
                <button class="btn outline-btn" onclick="editHost('${host.id}')" title="Edit" style="border: 1px solid rgba(255, 255, 255, 0.2); background: rgba(255,255,255,0.05);">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 6px; vertical-align: text-bottom;">
                        <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg> Edit
                </button>
                <button class="btn danger-btn" onclick="deleteHost('${host.id}', this)" title="Delete">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="vertical-align: text-bottom;">
                        <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2M10 11v6M14 11v6"/>
                    </svg>
                </button>
            </div>
        `;
        return card;
    };

    const escapeHTML = (str) => {
        const p = document.createElement('p');
        p.appendChild(document.createTextNode(str));
        return p.innerHTML;
    };

    // Load Hosts
    let loadHosts = async () => {
        try {
            const res = await apiFetch(`${apiBase}?t=${new Date().getTime()}`, {
                cache: 'no-store',
                headers: {
                    'Cache-Control': 'no-cache',
                    'Pragma': 'no-cache'
                }
            });
            if (!res.ok) throw new Error('Failed to load hosts');
            const hosts = await res.json();

            currentHosts = hosts || [];

            if (window.pingIntervals) {
                Object.values(window.pingIntervals).forEach(clearInterval);
            }
            window.pingIntervals = {};

            hostsContainer.innerHTML = '';

            if (!hosts || hosts.length === 0) {
                hostsContainer.innerHTML = '<div class="empty-state">No hosts found. Add one to get started!</div>';
                return;
            }

            window.startPing = (host) => {
                if (window.pingIntervals[host.id]) clearInterval(window.pingIntervals[host.id]);
                window.pingIntervals[host.id] = setInterval(() => {
                    apiFetch(`${apiBase}/${host.id}/ping`)
                        .then(r => r.json())
                        .then(data => {
                            if (!data.success && data.error === "Ping disabled") return;
                            const container = document.querySelector(`#ping-container-${host.id} .ping-track`);
                            if (!container) return;

                            const dot = document.createElement('div');
                            dot.className = `ping-dot ${data.success ? 'success' : 'error'}`;
                            container.appendChild(dot);

                            setTimeout(() => {
                                if (dot.parentNode === container) {
                                    container.removeChild(dot);
                                }
                            }, 12000);
                        })
                        .catch(() => { });
                }, 1000);
            };

            hosts.forEach(host => {
                hostsContainer.appendChild(createHostCard(host));

                if (host.ip && host.ping_enabled) {
                    window.startPing(host);
                }
            });
        } catch (err) {
            console.error(err);
            hostsContainer.innerHTML = '<div class="empty-state">Error loading hosts. Please ensure backend is running.</div>';
            showToast('Failed to load hosts', 'error');
        }
    };

    // Add Host
    addHostForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const btn = addHostForm.querySelector('button[type="submit"]');
        const origText = btn.textContent;
        btn.textContent = 'Saving...';
        btn.disabled = true;

        const newHost = {
            name: document.getElementById('host-name').value.trim(),
            mac_address: document.getElementById('host-mac').value.trim(),
            broadcast_ip: document.getElementById('host-ip').value.trim(),
            ip: document.getElementById('host-ping-ip').value.trim(),
            access_url: document.getElementById('host-access-url').value.trim(),
            ping_enabled: pingEnabledCheckbox.checked
        };

        const editingId = addHostForm.dataset.editingId;
        const method = editingId ? 'PUT' : 'POST';
        const url = editingId ? `${apiBase}/${editingId}` : apiBase;

        try {
            const res = await apiFetch(url, {
                method: method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(newHost)
            });

            if (!res.ok) {
                let errorMsg = editingId ? 'Failed to update host' : 'Failed to add host';
                const text = await res.text();
                if (text) errorMsg = text.trim();
                throw new Error(errorMsg);
            }

            addHostForm.reset();
            delete addHostForm.dataset.editingId;
            document.querySelector('#modal-title').textContent = 'Add New Host';
            addHostForm.querySelector('button[type="submit"]').textContent = 'Save Host';
            modalOverlay.classList.add('hidden');
            showToast(editingId ? 'Host updated successfully!' : 'Host added successfully!');
            await loadHosts();
        } catch (err) {
            showToast(err.message, 'error');
        } finally {
            btn.textContent = origText;
            btn.disabled = false;
        }
    });

    // Global Functions for inline onclick handlers
    window.editHost = (id) => {
        const host = currentHosts.find(h => h.id === id);
        if (!host) return;

        document.getElementById('host-name').value = host.name;
        document.getElementById('host-mac').value = host.mac_address || '';
        document.getElementById('host-ip').value = host.broadcast_ip || '';
        document.getElementById('host-ping-ip').value = host.ip || '';
        document.getElementById('host-access-url').value = host.access_url || '';
        pingEnabledCheckbox.checked = host.ping_enabled;

        addHostForm.dataset.editingId = id;

        document.querySelector('#modal-title').textContent = 'Edit Host';
        addHostForm.querySelector('button[type="submit"]').textContent = 'Update Host';

        modalOverlay.classList.remove('hidden');
        document.getElementById('host-name').focus();
    };

    window.wakeHost = async (id, btnElement) => {
        const origHtml = btnElement.innerHTML;
        btnElement.innerHTML = 'Sending...';
        btnElement.disabled = true;

        try {
            const res = await apiFetch(`${apiBase}/${id}/wake`, { method: 'POST' });
            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || 'Failed to send WOL');
            }
            showToast('Magic packet sent! ⚡');
        } catch (err) {
            showToast(err.message, 'error');
        } finally {
            btnElement.innerHTML = origHtml;
            btnElement.disabled = false;
            // Add a temporary success styling check
            btnElement.style.background = 'rgba(74, 222, 128, 0.2)';
            setTimeout(() => btnElement.style.background = '', 1000);
        }
    };

    window.toggleMonitoring = async (id, checkbox) => {
        const host = currentHosts.find(h => h.id === id);
        if (!host) {
            checkbox.checked = !checkbox.checked;
            return;
        }

        const originalState = host.ping_enabled;
        host.ping_enabled = checkbox.checked;

        // Optimistic UI update
        const container = document.querySelector(`#ping-container-${id}`);
        if (container) {
            container.style.opacity = !host.ping_enabled ? '0.3' : '1';
        }

        if (!host.ping_enabled) {
            if (window.pingIntervals[id]) {
                clearInterval(window.pingIntervals[id]);
                delete window.pingIntervals[id];
            }
        } else {
            if (host.ip) window.startPing(host);
        }

        try {
            const res = await apiFetch(`${apiBase}/${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(host)
            });
            if (!res.ok) throw new Error('Failed to update monitoring state');

            showToast(!host.ping_enabled ? 'Monitoring disabled' : 'Monitoring enabled');
        } catch (err) {
            showToast(err.message, 'error');
            // Revert
            checkbox.checked = originalState;
            host.ping_enabled = originalState;
            if (container) {
                container.style.opacity = !host.ping_enabled ? '0.3' : '1';
            }
            if (!host.ping_enabled) {
                if (window.pingIntervals[id]) {
                    clearInterval(window.pingIntervals[id]);
                    delete window.pingIntervals[id];
                }
            } else {
                if (host.ip) window.startPing(host);
            }
        }
    };

    window.deleteHost = async (id, btnElement) => {
        if (!confirm('Are you sure you want to delete this host?')) return;

        // Optimistic UI update
        const card = btnElement.closest('.host-card');
        card.style.opacity = '0.5';
        card.style.transform = 'scale(0.95)';

        try {
            const res = await apiFetch(`${apiBase}/${id}`, { method: 'DELETE' });
            if (!res.ok) throw new Error('Failed to delete host');

            showToast('Host deleted');
            await loadHosts();
        } catch (err) {
            showToast(err.message, 'error');
            card.style.opacity = '1';
            card.style.transform = 'none';
        }
    };

    // Master Toggle
    if (masterToggleCheckbox) {
        masterToggleCheckbox.addEventListener('change', async (e) => {
            const isEnabled = e.target.checked;

            // Optimistically update all cards visually
            document.querySelectorAll('.ping-container').forEach(container => {
                container.style.opacity = !isEnabled ? '0.3' : '1';
            });

            // Update internal state and toggles
            currentHosts.forEach(host => {
                host.ping_enabled = isEnabled;

                // Update specific host toggle if it exists in DOM
                const card = document.querySelector(`.host-card[data-id="${host.id}"]`);
                if (card) {
                    const toggle = card.querySelector('input[type="checkbox"]');
                    if (toggle) toggle.checked = isEnabled;
                }

                if (!isEnabled) {
                    if (window.pingIntervals[host.id]) {
                        clearInterval(window.pingIntervals[host.id]);
                        delete window.pingIntervals[host.id];
                    }
                } else if (host.ip) {
                    window.startPing(host);
                }
            });

            // Update backend for all hosts
            try {
                // Send concurrent requests for responsiveness, though a mass-update endpoint would be better
                await Promise.all(currentHosts.map(host =>
                    apiFetch(`${apiBase}/${host.id}`, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(host)
                    })
                ));
                showToast(isEnabled ? 'Monitoring enabled for all devices' : 'Monitoring disabled for all devices');
            } catch (err) {
                showToast('Failed to update some monitoring states', 'error');
                // Could refresh here to ensure sync with server
                await loadHosts();
            }

            updateMasterToggleState();
        });
    }

    const updateMasterToggleState = () => {
        if (!masterToggleCheckbox || currentHosts.length === 0) return;

        const anyMonitoring = currentHosts.some(h => h.ping_enabled);
        masterToggleCheckbox.checked = anyMonitoring;
    };

    // Call this after loadHosts completes
    const originalLoadHosts = loadHosts;
    loadHosts = async () => {
        await originalLoadHosts();
        updateMasterToggleState();
    };

    // Initial logic
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        const btn = loginForm.querySelector('button');
        const origText = btn.textContent;
        btn.textContent = 'Logging in...';
        btn.disabled = true;

        try {
            const res = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });

            if (!res.ok) throw new Error('Invalid credentials');

            const data = await res.json();
            authToken = data.token;
            localStorage.setItem('wakeonlan_token', authToken);
            hideLogin();
            showToast('Logged in successfully');
            loginForm.reset();
            loadHosts();
        } catch (err) {
            showToast(err.message, 'error');
        } finally {
            btn.textContent = origText;
            btn.disabled = false;
        }
    });

    if (!authToken) {
        showLogin();
    } else {
        loadHosts();
    }
});
