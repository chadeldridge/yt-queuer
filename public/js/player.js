// App variables.
let pbc = "";
let currentVideo = "";
let nextVideo = "";
let waitingForNextVideo = false;
const registerDiv = document.getElementById('register');
const registerForm = document.getElementById('registerForm');
const resultsDiv = document.getElementById('results');
const COOKIE_NAME = 'ytqueuer-playback_client';

// YouTube IFrame Player API variables.
let player, firstScriptTag, tag;
const playerDiv = document.getElementById('player');

function startup() {
        pbc = getCookie(COOKIE_NAME);
        if (pbc === null || pbc === "") {
                // Show the registration form and hide the player.
                showElement(registerDiv);
                hideElement(playerDiv);
                return
        }
        showPlayer();
}

function showPlayer() {
        // Insert the actual player iframe and hide the registration form. We use this iframe
        // method to ensure the player is always full window size.
        playerDiv.outerHTML = `<iframe id="player" type="text/html"
	src="https://www.youtube.com/embed/e4dmA-yJ4c0?enablejsapi=1"
	frameborder="0"
	style="width: 100%; height: 100%; position: absolute; top: 0; right: 0; bottom: 0; left: 0;">
</iframe>`
        hideElement(registerDiv);
        showElement(playerDiv);

        // Load the YouTube IFrame Player API code asynchronously.
        tag = document.createElement('script');
        tag.src = 'https://www.youtube.com/iframe_api';

        firstScriptTag = document.getElementsByTagName('script')[0];
        firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);

        // onYouTubeIframeAPIReady() will be called when the API is ready.
        // The created player will then call onPlayerReady() which will get the next video and
        // switch from the placeholder video to the first video in the queue and autoplay it.
}

function onYouTubeIframeAPIReady() {
        player = new YT.Player('player', {
                playerVars: {
                        'enablejsapi': 1,
                        'playsinline': 0,
                        'controls': 0,
                        'disablekb': 0,
                        'rel': 0,
                        'showinfo': 0,
                        'autoplay': 1,
                        'loop': 0,
                        'fs': 0,
                        'cc_load_policy': 0,
                        'iv_load_policy': 3,
                        'autohide': 1,
                        'origin': 'http://localhost:8080',
                },
                events: {
                        'onReady': onPlayerReady,
                        'onStateChange': onPlayerStateChange
                }
        });
}

const getPlaybackClientList = async () => {
        try {
                let list = '[]';
                const resp = await axios.get('/pbcs');
                if (resp.data != "") {
                        list = resp.data;
                }
        } catch(err) {
                handleFailure('Failed to get playback client list', err);
        }
}

function onPlayerReady(event) {
        playNextVideo();
}

function onPlayerStateChange(event) {
        if (event.data == YT.PlayerState.ENDED) {
                playNextVideo();
        }
}

function retryNextVideo() {
        waitingForNextVideo = false;
        getNextVideo();
}

const getNextVideo = async () => {
        try {
                if (waitingForNextVideo) {
                        return
                }

                waitingForNextVideo = false;
                const resp = await axios.get(`/playlists/${pbc.id}/next`);
                // If we do not get a 200 status code then we'll wait 2 seconds and try again.
                if (resp.status !== 200) {
                        currentVideo = resp.status;
                        // Calls getNextVideo() again after 2 seconds.
                        waitingForNextVideo = true;
                        window.setTimeout(retryNextVideo, 2000);
                        return
                }
        
                currentVideo = resp.data;
                player.loadVideoById(currentVideo.video_id, currentVideo.start_seconds);
        } catch(err) {
                currentVideo = err.status;
                handleFailure('Failed to get next video', err);
                // If we get an error then the service is probably down or there's some other
                // issue. We'll wait 10 seconds and try again.
                waitingForNextVideo = true;
                window.setTimeout(retryNextVideo, 10000);
        }
}

const peekNextVideo = async () => {
        try {
                const response = await axios.get(`/playlists/${pbc.id}/peek`);
                if (response.status === 204) {
                        nextVideo = 204;
                        return
                }
                
                nextVideo = response.data;
        } catch(err) {
                nextVideo = err.status;
                handleFailure('Failed to peek next video', err);
        }
}

const removeVideo = async (vid) => {
        try {
                const resp = await axios.delete(`/playlists/${pbc.id}/${vid}`);
                if (resp.status !== 204) {
                        log('failed to remove video:' + resp.data.message);
                        return
                }
        } catch(err) {
                handleFailure('Failed to remove video', err);
        }
}

const playNextVideo = async () => {
        try {
                // If we have valid video data then remove the video from the playlist.
                if (currentVideo !== null && currentVideo !== "" && currentVideo !== 204) {
                        await removeVideo(currentVideo.video_id);
                }

                // Get the next video and play it. If there are no videos in the playlist then
                // getNextVideo() will rerun itself until it gets one.
                await getNextVideo();
                // We should only get a 404 if the playback client is not found. Delete the cookie
                // and run startup() to show the registration form.
                if (currentVideo === 404) {
                        deleteCookie(COOKIE_NAME);
                        startup();
                        return
                }

                //if (currentVideo === 204) {
                //        await peekNextVideo();
                //}
        } catch(error) {
                handleFailure('Failed to play next video', error);
        }
};

registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(registerForm);
        const data = Object.fromEntries(formData.entries());

        try {
                if (data.name === null || data.name === "") {
                        log("name cannot be empty");
                        return
                }

                const resp = await axios.post('/pbcs/register?name=' + data.name);
                if (resp.status !== 200) {
                        log('failed to register playback client:' + resp.data.message);
                        return
                }

                log("playback client registered");
                setCookie(COOKIE_NAME, resp.data);
                
                startup();
        } catch(err) {
                handleFailure('Failed to register playback client', err);
        }
});


startup();
