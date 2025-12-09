const startBtn = document.getElementById("startBtn");
const cancelBtn = document.getElementById("cancelBtn");
const usernameInput = document.getElementById("username");
const passwordInput = document.getElementById("password");
const clientIdInput = document.getElementById("clientId");
const clientSecretInput = document.getElementById("clientSecret");
const inputFileEl = document.getElementById("inputFile");
const progressList = document.getElementById("progressList");
const queuePos = document.getElementById("queuePos");
const queueTotal = document.getElementById("queueTotal");
const queueInfo = document.getElementById("queueInfo");

let esProgress = null;
let esQueue = null;
let queuePoller = null;
let currentSessionId = null;

// helper to append progress lines
function appendProgress(text, cls) {
  const li = document.createElement("li");
  li.textContent = text;
  if (cls) li.className = cls;
  progressList.appendChild(li);
  li.scrollIntoView({ behavior: "smooth", block: "end" });
}

// Start import: send multipart form with file and credentials
async function startImport() {
  const username = usernameInput.value.trim();
  const password = passwordInput.value;
  const clientId = clientIdInput.value.trim();
  const clientSecret = clientSecretInput.value.trim();
  // Prefer explicit userId, otherwise fall back to clientId so user_id can be omitted
  const file = inputFileEl.files[0];

  if (!username || !password || !clientId || !clientSecret || !file) {
    alert(
      "Please provide username, password, client_id, client_secret and a CSV/XML file.",
    );
    return;
  }

  startBtn.disabled = true;
  appendProgress("Starting import...", "info");

  const fd = new FormData();
  fd.append("username", username);
  fd.append("password", password);
  fd.append("client_id", clientId);
  fd.append("client_secret", clientSecret);
  fd.append("manga_list", file, file.name);

  try {
    const resp = await fetch("/api/follow", {
      method: "POST",
      body: fd,
    });

    if (!resp.ok) {
      const text = await resp.text();
      appendProgress(`Error starting import: ${resp.status} ${text}`, "error");
      startBtn.disabled = false;
      return;
    }

    const data = await resp.json().catch(() => ({}));
    // prefer session_id returned by server, fall back to client_id or provided
    currentSessionId = data.session_id || data.clientId || null;
    if (!currentSessionId) {
      appendProgress(
        "No session id returned by server; cannot track progress.",
        "error",
      );
      startBtn.disabled = false;
      return;
    }

    appendProgress(`Import enqueued (session ${currentSessionId})`, "info");
    cancelBtn.disabled = false;
    queueInfo.hidden = false;

    // Start SSE progress stream using session_id
    startProgressSSE(currentSessionId);

    // Try subscribing to server queue SSE; if that fails, fallback to polling
    startQueueSSE() || startQueuePoll(currentSessionId);
  } catch (err) {
    appendProgress("Network error starting import", "error");
    console.error(err);
    startBtn.disabled = false;
  }
}

function startProgressSSE(sessionId) {
  if (esProgress) {
    esProgress.close();
    esProgress = null;
  }
  esProgress = new EventSource(
    `/api/progress?session_id=${encodeURIComponent(sessionId)}`,
  );

  esProgress.onopen = () =>
    appendProgress("Connected to progress stream", "info");
  esProgress.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data);
      if (data.type === "progress") {
        appendProgress(
          `${data.msg || "progress"} ${data.percent ? data.percent + "%" : ""}`,
          "progress",
        );
      } else if (data.type === "complete") {
        appendProgress("Import complete", "complete");
        cleanupAfterFinish();
      } else if (data.type === "error") {
        appendProgress(`Error: ${data.msg || "unknown"}`, "error");
        cleanupAfterFinish();
      } else {
        appendProgress(JSON.stringify(data), "info");
      }
    } catch (err) {
      appendProgress(e.data, "info");
    }
  };

  esProgress.onerror = (err) => {
    appendProgress("Progress stream disconnected.", "info");
  };
}

// Try to subscribe to a server-side queue SSE endpoint. Returns true if subscription established.
function startQueueSSE() {
  if (esQueue) {
    esQueue.close();
    esQueue = null;
  }
  try {
    esQueue = new EventSource(`/api/queue/subscribe`);
  } catch (err) {
    console.warn("queue SSE not available", err);
    return false;
  }

  esQueue.onopen = () => console.log("queue SSE connected");
  esQueue.onmessage = (e) => {
    try {
      const q = JSON.parse(e.data); // expected { queue_order: [...], queued: N }
      // If server returns queue_order and queued, find our position
      queueTotal.textContent = q.queued || q.queue_order.length || 0;
      if (currentSessionId && Array.isArray(q.queue_order)) {
        const pos = q.queue_order.indexOf(currentSessionId);
        queuePos.textContent = pos === -1 ? 0 : pos + 1;
      }
      // Optionally show a toast when position decreased (the user moved up)
      // We could track previous position and show a message when pos decreases
    } catch (err) {
      console.warn("invalid queue SSE message", err);
    }
  };
  esQueue.onerror = (err) => {
    console.warn("queue SSE closed or error", err);
    // If SSE fails, start fallback polling
    if (esQueue) {
      esQueue.close();
      esQueue = null;
    }
    if (currentSessionId) startQueuePoll(currentSessionId);
  };

  return true;
}

// Fallback: polling queue endpoint every 3s (if no SSE)
function startQueuePoll(sessionId) {
  if (queuePoller) clearInterval(queuePoller);
  queuePoller = setInterval(() => pollQueue(sessionId), 3000);
  pollQueue(sessionId);
}

async function pollQueue(sessionId) {
  try {
    const resp = await fetch(
      `/api/queue?session_id=${encodeURIComponent(sessionId)}`,
    );
    if (!resp.ok) return;
    const json = await resp.json();
    queuePos.textContent = json.position || 0;
    queueTotal.textContent = json.queued || 0;
  } catch (err) {
    console.warn("queue poll error", err);
  }
}

async function cancelImport() {
  if (!currentSessionId) return;
  const resp = await fetch(
    `/api/cancel?session_id=${encodeURIComponent(currentSessionId)}`,
    {
      method: "POST",
    },
  );
  if (resp.ok) appendProgress("Cancelled", "info");
  else appendProgress("Cancel request failed", "error");
  cleanupAfterFinish();
}

function cleanupAfterFinish() {
  if (esProgress) {
    esProgress.close();
    esProgress = null;
  }
  if (esQueue) {
    esQueue.close();
    esQueue = null;
  }
  if (queuePoller) {
    clearInterval(queuePoller);
    queuePoller = null;
  }
  cancelBtn.disabled = true;
  startBtn.disabled = false;
}

startBtn.addEventListener("click", startImport);
cancelBtn.addEventListener("click", cancelImport);
