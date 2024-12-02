const resLogSize = 4;
const menuItemIconClass = newClassList([
        'material-symbols-outlined',
        'text-lg',
        'mr-1',
        'text-neutral-100'
]);
const menuItemClass = newClassList([
        'block',
        'flex',
        'flex-row',
        'w-full',
        'whitespace-nowrap',
        'items-center',
        'px-4',
        'py-2',
        'leading-5',
        'cursor-pointer',
        'bg-neutral-800',
        'text-neutral-100',
        'hover:bg-neutral-700',
        'focus:outline-none',
        'focus:bg-neutral-500',
        'focus:text-neutral-900'
]);

let resLog = [];

function printResLog() {
        let resLogStr = "<div class='flex flex-col pl-3 pt-2 text-sm'>";
        for (let i = 0; i < resLog.length; i++) {
                let color = "";
                if (resLog.length - 1 == i) {
                        color = ' class="text-neutral-100"';
                }
                resLogStr += `<span${color}>${resLog[i]}</span>`;
        }
        resLogStr += "</div>";
        resultsDiv.innerHTML = resLogStr;
}

function addResLog(text) {
        resLog.push(text);
        if (resLog.length > resLogSize) {
                resLog.shift();
        }

        printResLog();
}

function log(text) {
        console.log(text);
        addResLog(text);
//        resultsDiv.innerHTML = `<h2 class="font-semibold">${text}</h2>`;
}

function handleFailure(msg, err) {
        if (err.response) {
                console.log(err.response);
                msg = `${msg}: (${err.response.status})` 
                if (err.response.data && err.response.data.message) {
                        msg += ` ${err.response.data.message}`;
                }
                log(msg);
        } else if (err.request) {
                log(`${msg}: No response received`);
        } else {
                log(`${msg}: ${err.message}`);
        }
}

// ############################################################################################## //
// ####################################   Cookie Functions   #################################### //
// ############################################################################################## //

function setCookie(name, value) {
        const date = new Date();
        date.setTime(date.getTime() + (365*24*60*60*1000));
        const expires = "expires=" + date.toUTCString();

        const data = JSON.stringify(value);
        document.cookie = `${name}=${data}; ${expires}; path=/; SameSite=None; secure`;
}

function getCookie(name) {
        let res = document.cookie.match(new RegExp(name + '=([^;]+)'));
        res && (res = JSON.parse(res[1]));
        return res;
}

function deleteCookie(name) {
        document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/; SameSite=None; secure`;
}

// ############################################################################################## //
// ####################################    HTML Functions    #################################### //
// ############################################################################################## //

// newElement creates a new element with the specified tag, text, and classes.
// Children should be an array of elements, a single element, string, or null.
// Classes should be a string (single class), an array of classes, a classList object, or null.
function newElement(tag, classes, children) {
        let e = document.createElement(tag);
        if (classes) {
                if (Array.isArray(classes)) {
                        classes.forEach((c) => { e.classList.add(c); });
                } else if (typeof classes === 'string') {
                        e.classList.add(classes);
                } else {
                        e.classList = classes;
                }
        }

        if (children) {
                e = addElements(e, children);
        }

        return e;
}

// addElements appends the children to the parent element.
// children should be an array of elements, a single element, or a string.
function addElements(parent, children) {
        if (Array.isArray(children)) {
                children.forEach((c) => { parent.appendChild(c); });
        } else if (typeof children === 'string') {
                parent.innerHTML += children;
        } else {
                parent.appendChild(children);
        }

        return parent;
}

// newClassList returns a classList object. If classes is an array, it adds them to the classList.
function newClassList(classes) {
        const e = document.createElement('div');
        if (classes && classes.length > 0) {
                classes.forEach((c) => { e.classList.add(c); });
        }

        return e.classList;
}

function toggleHidden(el) {
        el.classList.toggle("hidden");
}

function showElement(el) {
        if (el.classList.contains("hidden")) {
                el.classList.remove("hidden");
        }
}

function hideElement(el) {
        if (!el.classList.contains("hidden")) {
                el.classList.add("hidden");
        }
}
