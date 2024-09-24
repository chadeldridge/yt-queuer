let player, currentVideo, nextVideo;
let tag = document.createElement('script');
tag.src = 'https://www.youtube.com/iframe_api';

let firstScriptTag = document.getElementsByTagName('script')[0];
firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);

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

function onPlayerReady(event) {
        playNextVideo();
}

function onPlayerStateChange(event) {
        if (event.data == YT.PlayerState.ENDED) {
                playNextVideo();
        }
}

const getNextVideo = async () => {
        const response = await axios.get('/queue/next');
        if (response.status === 204) {
                currentVideo = 204;
                window.setTimeout(getNextVideo, 1000);
                return
        }
        
        currentVideo = response.data;
        player.loadVideoById(currentVideo.videoId, currentVideo.startSeconds);
}

const peekNextVideo = async () => {
        const response = await axios.get('/queue/peek');
        if (response.status === 204) {
                nextVideo = 204;
                return
        }
        
        nextVideo = response.data;
}

const playNextVideo = async () => {
        try {
                await getNextVideo();
                if (currentVideo === 204) {
                        return
                }

                // Peek the next video
                await peekNextVideo();
                if (nextVideo.status === 204) {
                        return
                }
        } catch(error) {
                console.error(error);
                errorDiv.innerHTML = error.response.data.message;
        }
};
