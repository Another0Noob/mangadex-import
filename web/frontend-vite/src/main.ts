import "./style.css";

type Mode = "import" | "export";

// State
let currentMode: Mode = "import";

// Elements
const modeButtons = document.querySelectorAll(
  ".mode-btn",
) as NodeListOf<HTMLButtonElement>;
const submitBtn = document.getElementById("submitBtn") as HTMLButtonElement;
const resultDiv = document.getElementById("result") as HTMLDivElement;
const resultText = document.getElementById("resultText") as HTMLDivElement;

// Form fields
const usernameField = document.querySelector(
  '[data-field="username"]',
) as HTMLDivElement;
const passwordField = document.querySelector(
  '[data-field="password"]',
) as HTMLDivElement;
const client_idField = document.querySelector(
  '[data-field="client_id"]',
) as HTMLDivElement;
const client_secretField = document.querySelector(
  '[data-field="client_secret"]',
) as HTMLDivElement;
const mangaField = document.querySelector(
  '[data-field="manga"]',
) as HTMLDivElement;

// Inputs
const usernameInput = document.getElementById("username") as HTMLInputElement;
const passwordInput = document.getElementById("password") as HTMLInputElement;
const client_idInput = document.getElementById("client_id") as HTMLInputElement;
const client_secretInput = document.getElementById(
  "client_secret",
) as HTMLInputElement;
const mangaInput = document.getElementById("manga") as HTMLInputElement;

// Field visibility rules
const fieldRules: Record<Mode, string[]> = {
  import: ["username", "password", "client_id", "client_secret", "manga"],
  export: ["username", "password", "client_id", "client_secret"],
};

// Update UI based on mode
function updateUI(mode: Mode): void {
  currentMode = mode;

  // Update mode buttons
  modeButtons.forEach((btn) => {
    btn.classList.remove("active-import", "active-export");
    if (btn.dataset.mode === mode) {
      btn.classList.add(`active-${mode}`);
    }
  });

  // Show/hide fields based on mode
  const visibleFields = fieldRules[mode];
  [
    passwordField,
    usernameField,
    client_idField,
    client_secretField,
    mangaField,
  ].forEach((field) => {
    const fieldName = field.dataset.field!;
    if (visibleFields.includes(fieldName)) {
      field.classList.add("visible");
    } else {
      field.classList.remove("visible");
    }
  });

  // Update submit button
  submitBtn.className = `submit-btn ${mode}`;
  submitBtn.textContent = mode === "import" ? "Import Manga" : "Update Entry";

  // Hide result when mode changes
  resultDiv.classList.remove("visible");
}

// Handle mode button clicks
modeButtons.forEach((btn) => {
  btn.addEventListener("click", () => {
    const mode = btn.dataset.mode as Mode;
    updateUI(mode);
  });
});

// Handle submit
submitBtn.addEventListener("click", () => {
  let message = "";

  switch (currentMode) {
    case "import":
      break;
    case "export":
      break;
  }

  resultText.textContent = message;
  resultDiv.classList.add("visible");
});

// Initialize
updateUI("import");
