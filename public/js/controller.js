const PLAYLIST_RETRY = 10000;
const COOKIE_NAME = 'ytqueuer-controller';
const pbcMenu = document.getElementById('pbcs');
const selectedPBC = document.getElementById('selectedPBC');
const resultsDiv = document.getElementById('results');
const playlistDiv = document.getElementById('playlist');
const powerSettingsMenu = document.getElementById('powerSettingsMenu');
const powerSettingsDiv = document.getElementById('powerSettings');
const powerSettingsDetails = document.getElementById('powerSettingsDetails');
const cecSettingsForm = document.getElementById('cecSettings');
const wolConfigDiv = document.getElementById('wolConfig');
const wolSettingsForm = document.getElementById('wolSettings')
const psdCECTab = document.getElementById('psdCECTab');
const psdWOLTab = document.getElementById('psdWOLTab');

let psdActive = "";
let psdActiveTab = "";
let currentPlaylist = "";
let resultsCountdown = "";
let resultsTimeoutDiv = "";
let powerSettinsMenuOpen = false;
let wol = "";
let cec = "";
let waitingForPlaylists = false;
let playlists = [];
let playlist = [];

function retryPlaylistsWatcher() {
        waitingForPlaylists = false;
        playlistsWatcher();
}

const playlistsWatcher = async () => {
        // Name the try block so we can break out of it if needed.
        playlist: try {
                // If we're already waiting for playlists, return.
                if (waitingForPlaylists) {
                        return
                }

                playlists = await getPlaybackClients();
                if (playlists === null) {
                        log('Failed to get playlists.');
                        // Break out of the playlist try block.
                        break playlist;
                }

                // If we got a list of playlists, fill the dropdown.
                fillPlaylists();

                if (currentPlaylist !== null && currentPlaylist !== "") {
                        getPlaylist();
                }
        } catch(err) {
                handleFailure('Failed to get playlists', err);
        } finally {
                window.setTimeout(retryPlaylistsWatcher, PLAYLIST_RETRY);
        }
}

const getClipboardData = async () => {
        try {
                const text = await navigator.clipboard.readText();
                if (!text.includes('youtube.com/watch?v=')) {
                        throw new Error('Invalid URL in clipboard.');
                }

                id = text.split("v=")[1];
                if (text.includes('&')) {
                        parts = id.split('&');
                        return parts[0];
                }

                return id;
        } catch(err) {
                // This is not an axios call so we can't use handleFailure().
                log(`Failed to get data from clipboard: '${err}'`);
        }
}

const getPlaybackClients = async () => {
        try {
                const resp = await axios.get('/pbcs');
                let list = [];

                if (resp.data != "") {
                        list = resp.data;
                }

                return list;
        } catch(err) {
                handleFailure('Failed to get available playlists', err);
                return null;
        }
}

const getPlaylist = async () => {
        try {
                const resp = await axios.get('/playlists/' + currentPlaylist.id);
                if (resp.status !== 200 && resp.status !== 204) {
                        log(`Failed to get playlists: (${resp.status}) ${resp.data.message}`);
                        return
                }
        
                showPlaylist(resp.data);
        } catch(err) {
                handleFailure(`Failed to get playlist for '${currentPlaylist.name}'`, err);
        }
}

// Log a message if a playlist is not selected and return false. Otherwise, return true.
function IsPlaylistSelected() {
        if (currentPlaylist === null || currentPlaylist === "" || currentPlaylist === "{}") {
                log("You must select a playlist first.");
                return false
        }

        return true
}

const addVideo = async () => {
        try {
                if (!IsPlaylistSelected()) {
                        return
                }

                const videoID = await getClipboardData();
                if (!videoID) {
                        return;
                }
        
                const resp = await axios.post(`/playlists/${currentPlaylist.id}/${videoID}`);
                log(resp.data.message);
                getPlaylist();
        } catch(err) {
                handleFailure('Failed to add video to playlist', err);
        }
}

const addNext = async () => {
        try {
                if (!IsPlaylistSelected()) {
                        return
                }

                const videoID = await getClipboardData();
                if (!videoID) {
                        return;
                }
        
                const resp = await axios.post(`/playlists/${currentPlaylist.id}/${videoID}/next`);
                log(resp.data.message);
                getPlaylist();
        } catch(err) {
                handleFailure('Failed to add video to top of playlist', err);
        }
}

const removeVideo = async (vid) => {
        try {
                if (!IsPlaylistSelected()) {
                        return
                }

                const resp = await axios.delete(`/playlists/${currentPlaylist.id}/${vid}`);
                if (resp.status !== 204) {
                        log('failed to remove video:' + resp.data.message);
                        return
                }
        
                log("video removed");
                getPlaylist();
        } catch(err) {
                handleFailure('Failed to remove video from playlist', err);
        }
}

const clearPlaylist = async () => {
        try {
                if (!IsPlaylistSelected()) {
                        return
                }

                const resp = await axios.delete('/playlists/' + currentPlaylist.id);
                if (resp.status !== 204) {
                        log('failed to clear playlist:' + resp.data.message);
                        return
                }
        
                log("playlist cleared");
                showPlaylist();
        } catch(err) {
                handleFailure('Failed to clear playlist', err);
        }
}

function playlistSelected() {
        if (currentPlaylist === null || currentPlaylist === "" || currentPlaylist.id === "") {
                return false
        }

        return true
}

function showPlaylist(q) {
        playlistDiv.innerHTML = "";

        if (!q || q === "" || q.length === 0) {
                playlistDiv.appendChild(
                        newElement(
                                'div',
                                ['w-full', 'h-full', 'text-center', 'content-center', 'font-bold', 'text-4xl', 'text-neutral-700'],
                                'Playlist Empty'
                        )
                );

                return
        }

        let ul = newElement('ul', ['pt-3', 'bg-neutral-600', 'rounded-xl']);
        q.forEach((v) => {
                ul.innerHTML +=
`<li>
        <div class="flex flex-row justify-between items-center pb-3">
                <div class="flex flex-row">
                        <div><img src="${v.thumbnail_url}" style="width:120px;height:90px"></div>
                        <div class="flex flex-col pl-6">
                                <div>${v.title}</div>
                                <div>${v.author_name}</div>
                        </div>
                </div>
                <div><button type="button" class="material-symbols-outlined text-5xl p-3 pl-6 hover:text-secondary-base" title="Remove Video" onClick="removeVideo('${v.video_id}')">playlist_remove</button></div>
        </div>
</li>`;
        });
        playlistDiv.appendChild(ul);
}

function fillPlaylists() {
        // Clear the list so we can rebuild it.
        pbcMenu.innerHTML = "";

        // If there are no playlists, display a message.
        if (playlists.length === 0) {
                selectedPBC.innerHTML = "No Playback Clients Available";
                currentPlaylist = "";
                return
        }

        // If no playlist is selected, display a message.
        if (currentPlaylist === null || currentPlaylist === "") {
                selectedPBC.innerHTML = "Select a Playlist";
                currentPlaylist = "";
        }

        // Fill the dropdown with the playlists.
        for (let i = 0; i < playlists.length; i++) {
                let li = newElement('li', menuItemClass, newElement('span', null, playlists[i].name));
                li.value = i;

                // Round the corners of the first and last items.
                if (playlists.length === 1) {
                        li.classList.add('rounded-xl');
                } else if (i === 0) {
                        li.classList.add('rounded-t-xl');
                } else if (i === playlists.length - 1) {
                        li.classList.add('rounded-b-xl');
                }

                pbcMenu.appendChild(li);
        }
}

function selectPBC(e) {
        value = e.target.value;
        if (value >= playlists.length) {
                log("Invalid playlist selected.");
                return
        }

        currentPlaylist = playlists[value];
        selectedPBC.innerHTML = currentPlaylist.name;

        setCookie(COOKIE_NAME, currentPlaylist);
        getPlaylist();
        updatePowerSettingsMenu();
}

// ############################################################################################## //
// ####################################    Power Settings    #################################### //
// ############################################################################################## //

function psdTabSelect(tab) {
        // Unhighlight the active tab.
        if (psdActiveTab !== "") {
                psdActiveTab.classList.remove('bg-neutral-700');
                psdActiveTab.classList.add('bg-neutral-800');
        }

        // Highlight the selected tab.
        psdActiveTab = tab;
        tab.classList.remove('bg-neutral-800');
        tab.classList.add('bg-neutral-700');
}

function deletePowerSettings(setting) {
        hidePowerSettings();
        if (setting === "wol") {
                deleteWOL();
        } else if (setting === "cec") {
                deleteCEC();
        }

        updatePowerSettingsMenu();
}

function switchPowerSettings(form, tab) {
        psdTabSelect(tab);
        if (psdActive !== "") {
                hideElement(psdActive);
        }

        powerSettingsDetails.innerHTML = "";
        showElement(form);
        powerSettingsDetails.appendChild(form);

        psdActive = form;
}

function showPowerSettings() {
        if (playlistSelected() === false) {
                log("You must select a playlist first.");
                return
        }

        powerSettinsMenuOpen = true;
        showElement(powerSettingsDiv);
}

function hidePowerSettings() {
        powerSettinsMenuOpen = false;
        hideElement(powerSettingsDiv);
}

const updatePowerSettingsMenu = async () => {
        // Hide the Wake On LAN menu if no playlist is selected.
        if (currentPlaylist === null || currentPlaylist === "") {
                log("You must select a playlist first.");
                hideElement(powerSettingsMenu);
                return
        }

        // Get CEC and Wake On LAN settings for the current playlist.
        await getCEC();
        updateCECSettings();

        await getWOL();
        updateWOLSettings();


        // Clear the menu so we can rebuild it.
        powerSettingsMenu.innerHTML = "";

        if (wol !== null && wol !== "") {
                // Create a LAN icon to use for Wake On LAN menu items.
                let wolIcon = newElement('span', menuItemIconClass, 'lan');

                // Add the Wake On LAN menu item.
                let wolItem = newElement('li', menuItemClass, [
                        wolIcon,
                        newElement('span', null, wol.alias)
                ]);
                wolItem.onmousedown = function() { wolWake() };
                powerSettingsMenu.appendChild(wolItem);
        }

        if (cec !== null && cec !== "") {
                // Create a CEC icon to use for CEC menu items.
                let cecIcon = document.createElement('span');
                cecIcon.classList.add(
                        'material-symbols-outlined',
                        'text-lg',
                        'mr-1',
                        'text-neutral-100'
                );
                cecIcon.innerHTML = "settings_input_hdmi";

                // Add Power Status menu item.
                let cecStatus = newElement('li', menuItemClass, [
                        cecIcon.cloneNode(true),
                        newElement('span', null, 'Power Status')
                ]);
                cecStatus.onmousedown = function() { cecPowerStatus() };
                powerSettingsMenu.appendChild(cecStatus);

                // Add Power On menu item.
                let cecOn = newElement('li', menuItemClass, [
                        cecIcon.cloneNode(true),
                        newElement('span', null, 'Power On')
                ]);
                cecOn.onmousedown = function() { cecPowerOn() };
                powerSettingsMenu.appendChild(cecOn);

                // Add Power Off menu item.
                let cecOff = newElement('li', menuItemClass, [
                        cecIcon.cloneNode(true),
                        newElement('span', null, 'Power Off')
                ]);
                cecOff.onmousedown = function() { cecPowerOff() };
                powerSettingsMenu.appendChild(cecOff);
        }

        // Add the Edit Power Settings option.
        let epsItem = document.createElement('li');
        epsItem.classList = menuItemClass;
        epsItem.onmousedown = function() { showPowerSettings() };
        epsItem.innerHTML = "Edit Power Settings";
        powerSettingsMenu.appendChild(epsItem);
}

const updateCECSettings = async () => {
        // Don not update the form if it is open.
        if (powerSettinsMenuOpen) {
                return
        }

        if (cec === null || cec === "") {
                cecSettingsForm.alias.value = "";
                cecSettingsForm.device.value = "";
                cecSettingsForm.logical_addr.value = "";
                cecSettingsForm.physical_addr.value = "";
                return
        }

        // Only update empty input fields.
        cecSettingsForm.alias.value = cec.alias;
        cecSettingsForm.device.value = cec.device;
        cecSettingsForm.logical_addr.value = cec.logical_addr;
        cecSettingsForm.physical_addr.value = cec.physical_addr;
}

const updateWOLSettings = async () => {
        // Don not update the form if it is open.
        if (powerSettinsMenuOpen) {
                return
        }

        if (wol === null || wol === "") {
                wolSettingsForm.alias.value = "";
                wolSettingsForm.iface.value = "";
                wolSettingsForm.mac.value = "";
                wolSettingsForm.port.value = "";
                return
        }

        wolSettingsForm.alias.value = wol.alias;
        wolSettingsForm.iface.value = wol.iface;
        wolSettingsForm.mac.value = wol.mac;
        wolSettingsForm.port.value = wol.port;
}

// ########## WOL ########## //

const createWOL = async (data) => {
        try {
                uri = `/wol/${currentPlaylist.id}?alias=${data.alias}&iface=${data.iface}&mac=${data.mac}&port=${data.port}`;
                const resp = await axios.post(uri);
                if (resp.status !== 201) {
                        log('failed to create WOL:' + resp.data.message);
                }

                return resp.status;
        } catch(err) {
                handleFailure('Failed to create Wake On LAN', err);
        }
}

const getWOL = async () => {
        try {
                const resp = await axios.get(`/wol/${currentPlaylist.id}`);
                if (resp.status !== 200) {
                        //log('failed to get WOL:' + resp.data.message);
                        return
                }

                wol = resp.data;
        } catch(err) {
                handleFailure('Failed to get Wake On LAN', err);
        }
}

const updateWOL = async (data) => {
        try {
                uri = `/wol/${currentPlaylist.id}?alias=${data.alias}&iface=${data.iface}&mac=${data.mac}&port=${data.port}`;
                const resp = await axios.put(uri);
                if (resp.status !== 200) {
                        log('failed to update WOL:' + resp.data.message);
                }

                return resp.status;
        } catch(err) {
                handleFailure('Failed to update Wake On LAN', err);
        }
}

const deleteWOL = async () => {
        try {
                const resp = await axios.delete('/wol/' + currentPlaylist.id);
                if (resp.status !== 204) {
                        log('failed to delete WOL:' + resp.data.message);
                        return
                }

                updatePowerSettingsMenu();
        } catch(err) {
                handleFailure('Failed to delete Wake On LAN', err);
        }
}

const wolWake = async () => {
        try {
                log(`Requesting Wake On Lan for ${currentPlaylist.name}...`);
                const resp = await axios.post(`/wol/${currentPlaylist.id}/wake`);
                if (resp.status !== 200) {
                        log('failed to send WOL:' + resp.data.message);
                        return
                }

                log(`Wake On Lan sent to ${currentPlaylist.name}.`);
        } catch(err) {
                handleFailure('Failed to send Wake On LAN', err);
        }
}

// ########## CEC ########## //

const createCEC = async (data) => {
        try {
                uri = `/cec/${currentPlaylist.id}?alias=${data.alias}&device=${data.device}&logical_addr=${data.logical_addr}&physical_addr=${data.physical_addr}`;
                const resp = await axios.post(uri);
                if (resp.status !== 201) {
                        log('failed to create CEC:' + resp.data.message);
                }

                cec = resp.data;
                return resp.status;
        } catch(err) {
                handleFailure('Failed to create CEC', err);
        }
}

const getCEC = async () => {
        try {
                const resp = await axios.get(`/cec/${currentPlaylist.id}`);
                if (resp.status !== 200) {
                        //log('failed to get CEC:' + resp.data.message);
                        return
                }

                cec = resp.data;
        } catch(err) {
                handleFailure('Failed to get CEC', err);
        }
}

const updateCEC = async (data) => {
        try {
                uri = `/cec/${currentPlaylist.id}?alias=${data.alias}&device=${data.device}&logical_addr=${data.logical_addr}&physical_addr=${data.physical_addr}`;
                const resp = await axios.put(uri);
                if (resp.status !== 200) {
                        log('failed to update CEC:' + resp.data.message);
                }

                return resp.status;
        } catch(err) {
                handleFailure('Failed to update CEC', err);
        }
}

const deleteCEC = async () => {
        try {
                const resp = await axios.delete('/cec/' + currentPlaylist.id);
                if (resp.status !== 204) {
                        log('failed to delete CEC:' + resp.data.message);
                        return
                }
        } catch(err) {
                handleFailure('Failed to delete CEC', err);
        }
}

const cecPowerOn = async () => {
        try {
                log(`Requesting CEC Power On for ${currentPlaylist.name}...`);
                const resp = await axios.post(`/cec/${currentPlaylist.id}/power/on`);
                if (resp.status !== 200) {
                        log('failed to send CEC Power On:' + resp.data.message);
                        return
                }

                log(`CEC Power On sent to ${currentPlaylist.name}.`);
        } catch(err) {
                handleFailure('Failed to send CEC Power On', err);
        }
}

const cecPowerOff = async () => {
        try {
                log(`Requesting CEC Power Off for ${currentPlaylist.name}...`);
                const resp = await axios.post(`/cec/${currentPlaylist.id}/power/off`);
                if (resp.status !== 200) {
                        log('failed to send CEC Power Off:' + resp.data.message);
                        return
                }

                log(`CEC Power Off sent to ${currentPlaylist.name}.`);
        } catch(err) {
                handleFailure('Failed to send CEC Power Off', err);
        }
}

const cecPowerStatus = async () => {
        try {
                log(`Requesting CEC Power Status for ${currentPlaylist.name}...`);
                const resp = await axios.get(`/cec/${currentPlaylist.id}/power/status`);
                if (resp.status !== 200) {
                        log('failed to get CEC Power Status:' + resp.data.message);
                        return
                }

                log(`CEC Power Status for ${currentPlaylist.name}: ${resp.data.power}`);
        } catch(err) {
                handleFailure('Failed to get CEC Power Status', err);
        }
}

// ############################################################################################## //
// ####################################      Listeners       #################################### //
// ############################################################################################## //

pbcMenu.addEventListener('mousedown', async (e) => { selectPBC(e) });

cecSettingsForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(cecSettingsForm);
        const data = Object.fromEntries(formData.entries());
        hidePowerSettings();

        if (data === null || data.length === 0) {
                log('No CEC settings provided. Form was empty.');
                return
        }

        if (cec === null || cec === "") {
                const status = await createCEC(data);
                if (status !== 201) {
                        return
                }
        } else {
                const status = await updateCEC(data);
                if (status !== 200) {
                        return
                }
        }

        updatePowerSettingsMenu();
});

wolSettingsForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(wolSettingsForm);
        const data = Object.fromEntries(formData.entries());
        hidePowerSettings();

        if (wol === null || wol === "") {
                const status = await createWOL(data);
                if (status !== 201) {
                        return
                }
        } else {
                const status = await updateWOL(data);
                if (status !== 200) {
                        return
                }
        }

        updatePowerSettingsMenu();
});

// ############################################################################################## //
// ####################################       Startup        #################################### //
// ############################################################################################## //

const startup = async () => {
        // Get the current playlist from the cookie.
        let cp = getCookie(COOKIE_NAME);
        currentPlaylist = cp;
        if (currentPlaylist === null || currentPlaylist === "") {
                return
        }

        pbcMenu.value = currentPlaylist.id;
        await getPlaylist();
        if (playlists === null || playlists === "" || playlists.length === 0) {
                currentPlaylist = "";
        }

        updatePowerSettingsMenu();
}

switchPowerSettings(cecSettingsForm, psdCECTab);
// Start the controller.
startup();

// Load the playlists when the page loads.
playlistsWatcher();
