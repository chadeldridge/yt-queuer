const resultsDiv = document.getElementById('results');
const queueDiv = document.getElementById('queue');

function log(text) {
        resultsDiv.innerHTML = text;
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
                log(err);
        }
}

const getQueue = async () => {
        const resp = await axios.get('/queue');
        if (resp.status !== 200) {
                if (resp.status === 204) {
                        showQueue();
                        return
                }
                resultsDiv.innerHTML = 'failed to get queue:' + resp.data.message;
                return
        }

        showQueue(resp.data);
}

const addVideo = async () => {
        const videoID = await getClipboardData();
        if (!videoID) {
                return;
        }

        const resp = await axios.get('/queue/add/' + videoID);
        resultsDiv.innerHTML = resp.data.message;
        getQueue();
}

const addNext = async () => {
        const videoID = await getClipboardData();
        if (!videoID) {
                return;
        }

        const resp = await axios.get('/queue/playnext/' + videoID);
        resultsDiv.innerHTML = resp.data.message;
        getQueue();
}

const clearQueue = async () => {
        const resp = await axios.get('/queue/clear');
        if (resp.status !== 204) {
                resultsDiv.innerHTML = 'failed to clear queue:' + resp.data.message;
                return
        }

        resultsDiv.innerHTML = "queue cleared";
        showQueue();
}

const removeVideo = async (vid) => {
        console.log("Removing video: " + vid);
        const resp = await axios.get('/queue/remove/' + vid);
        if (resp.status !== 204) {
                resultsDiv.innerHTML = 'failed to remove video:' + resp.data.message;
                return
        }

        resultsDiv.innerHTML = "video removed";
        getQueue();
}

function showQueue(q) {
        if (!q || q.length === 0) {
                queueDiv.innerHTML = "Queue is empty.";
                return
        }

        let c = '<ul class="pt-3">';
        q.forEach((v) => {
                c +=
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
        c += "</ul>";
        queueDiv.innerHTML = c;
}

// Load the queue when the page loads
getQueue();
