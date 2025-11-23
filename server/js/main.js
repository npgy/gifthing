let isValidUrl = false;

// URL validation function
function validateGifUrl(url) {
  try {
    const urlObj = new URL(url);

    const validExtensions = [".gif", ".mp4"];

    // Check if it's a valid URL
    if (!url || !urlObj.protocol.startsWith("http")) {
      return "Please enter a valid URL";
    }

    const hasValidExtension = validExtensions.some((ext) =>
      urlObj.pathname.toLowerCase().includes(ext),
    );

    if (!hasValidExtension) {
      return "URL must be a gif or mp4 file";
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
  goButton.disabled = true;
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

  let res = await fetch("/setgif", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ gifUrl: url }),
  });

  console.log(await res.json());

  goButton.disabled = false;
  loading.classList.remove("show");
}

// Go button click event
goButton.addEventListener("click", async () => await sendToGifthing());

// Focus input on page load
window.addEventListener("load", () => {
  gifInput.focus();
});
