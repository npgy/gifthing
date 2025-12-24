let isValidUrl = false;

// Helper function to check if URL is a Tenor URL
function isTenorUrl(url) {
  try {
    const urlObj = new URL(url);
    return urlObj.hostname.includes("tenor.com");
  } catch (e) {
    return false;
  }
}

// URL validation function
function validateGifUrl(url) {
  try {
    const urlObj = new URL(url);

    const validExtensions = [".gif", ".mp4"];

    // Check if it's a valid URL
    if (!url || !urlObj.protocol.startsWith("http")) {
      return "Please enter a valid URL";
    }

    // Check if it's a Tenor URL
    if (urlObj.hostname.includes("tenor.com")) {
      return null; // Tenor URLs are valid
    }

    const hasValidExtension = validExtensions.some((ext) =>
      urlObj.pathname.toLowerCase().includes(ext)
    );

    if (!hasValidExtension) {
      return "URL must be a Tenor, gif, or mp4 url";
    }

    return null;
  } catch (e) {
    return "Please enter a valid URL";
  }
}

function validateAndShowError(url) {
  const error = validateGifUrl(url);
  if (error) {
    showError(error);
  } else {
    hideError();
    isValidUrl = true;
    goButton.disabled = false;
  }
}

function showError(message) {
  errorMessage.textContent = message;
  errorMessage.classList.add("show");
  gifInput.classList.add("error");
  isValidUrl = false;
}

function hideError() {
  errorMessage.classList.remove("show");
  errorMessage.textContent = "";
  gifInput.classList.remove("error");
}

gifInput.addEventListener("input", (e) => {
  const url = e.target.value.trim();

  hideError();

  if (!url) {
    isValidUrl = false;
    goButton.disabled = true;
    return;
  }

  validateAndShowError(url);
});

// Handle paste events
gifInput.addEventListener("paste", (e) => {
  setTimeout(() => {
    const url = gifInput.value.trim();
    if (url) {
      validateAndShowError(url);
    }
  }, 10);
});

// Handle Enter key
gifInput.addEventListener("keypress", async (e) => {
  if (e.key === "Enter" && isValidUrl) {
    await sendToGifthing();
  }
});

// Process URL function
async function sendToGifthing() {
  const url = gifInput.value.trim();

  if (!isValidUrl || !url) return;

  // Show loading state
  goButton.disabled = true;
  loading.classList.add("show");

  try {
    let finalUrl = url;

    // If it's a Tenor URL, get the MP4 URL first
    if (isTenorUrl(url)) {
      const tenorRes = await fetch("/tenor-mp4", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ tenorGifUrl: url }),
      });

      if (!tenorRes.ok) {
        throw new Error(`Failed to get MP4 URL from Tenor: ${tenorRes.status}`);
      }

      const tenorData = await tenorRes.json();
      finalUrl = tenorData.mp4Url;
      console.log("Got MP4 URL from Tenor:", finalUrl);
    }

    // Now call setgif with the final URL (either original or converted MP4)
    let res = await fetch("/setgif", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        gifUrl: finalUrl,
        isTenor: isTenorUrl(url),
      }),
    });

    if (!res.ok) {
      throw new Error(`Failed to set GIF: ${res.status}`);
    }

    console.log("Successfully sent to gifthing");

    // Refresh previous gifs list
    setTimeout(() => loadPreviousGifs(), 500);
  } catch (error) {
    console.error("Error processing URL:", error);
    showError("Failed to process URL. Please try again.");
  }

  goButton.disabled = false;
  loading.classList.remove("show");
}

// Go button click event
goButton.addEventListener("click", async () => await sendToGifthing());

// Focus input on page load
window.addEventListener("load", () => {
  gifInput.focus();
  loadPreviousGifs();
});

// Load and display previous gifs
async function loadPreviousGifs() {
  try {
    const response = await fetch("/previous-gifs");
    if (!response.ok) {
      console.error("Failed to load previous gifs:", response.status);
      return;
    }

    const gifs = await response.json();
    displayPreviousGifs(gifs);
  } catch (error) {
    console.error("Error loading previous gifs:", error);
  }
}

// Display previous gifs in the grid
function displayPreviousGifs(gifs) {
  const grid = document.getElementById("previousGrid");

  if (!gifs || gifs.length === 0) {
    grid.innerHTML =
      '<div style="grid-column: 1 / -1; text-align: center; color: #666; padding: 2rem;">No previous gifs yet</div>';
    return;
  }

  grid.innerHTML = gifs
    .map((gif) => {
      const previewUrl = gif.previewUrl || gif.url;
      const isVideo = previewUrl.toLowerCase().endsWith(".mp4");

      if (isVideo) {
        return `
    <div class="previous-item" onclick="loadPreviousGif('${gif.url}')" title="Click to load this gif">
      <video src="${previewUrl}" loop muted playsinline onerror="this.style.display='none'"></video>
    </div>`;
      } else {
        return `
    <div class="previous-item" onclick="loadPreviousGif('${gif.url}')" title="Click to load this gif">
      <img src="${previewUrl}" alt="Previous gif" onerror="this.style.display='none'" />
    </div>`;
      }
    })
    .join("");
}

// Load a previous gif when clicked
async function loadPreviousGif(url) {
  gifInput.value = url;
  validateAndShowError(url);

  // Auto-submit if valid
  if (isValidUrl) {
    await sendToGifthing();
  }
}
