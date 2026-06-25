describe("QPU Registry — Admin: Register QPU", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    // Log in as admin
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // Navigate to QPU Registry
    cy.contains("button", "QPU Registry").click();
    cy.contains("h1", "QPU Registry").should("be.visible");
  });

  it("opens the registration modal and shows the form", () => {
    cy.contains("button", "Register QPU").click();

    cy.contains("h3", "Register QPU").should("be.visible");
    cy.get('input[placeholder="rigetti-aspen-9"]').should("be.visible");
    cy.get("select").should("be.visible");
    cy.contains("button", "Register Unit").should("be.visible");
  });

  it("registers a QPU and shows the success screen with correct details", () => {
    cy.contains("button", "Register QPU").click();

    const qpuName = `cypress-test-qpu-${Date.now()}`;
    const executor = "qblox";

    cy.get('input[placeholder="rigetti-aspen-9"]').type(qpuName);
    cy.get("select").select(executor);
    cy.contains("button", "Register Unit").click();

    // Success screen appears
    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");

    // The exact name and executor entered are displayed
    cy.contains(`Your QPU ${qpuName} has been registered.`).should(
      "be.visible",
    );
    cy.contains(executor).should("be.visible");
  });

  it("shows a non-empty access token on the success screen", () => {
    cy.contains("button", "Register QPU").click();

    cy.get('input[placeholder="rigetti-aspen-9"]').type(`token-test-${Date.now()}`);
    cy.get("select").select("mock");
    cy.contains("button", "Register Unit").click();

    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");

    // Access Token section is visible
    cy.contains("label", "Access Token").should("be.visible");

    // The token value is a non-empty string
    cy.get("span.font-mono")
      .contains(/\S+/)
      .should("be.visible");
  });

  it("shows a connection command with the required flags", () => {
    cy.contains("button", "Register QPU").click();

    const qpuName = `cmd-test-${Date.now()}`;
    cy.get('input[placeholder="rigetti-aspen-9"]').type(qpuName);
    cy.get("select").select("qiskit_aer");
    cy.contains("button", "Register Unit").click();

    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");

    // Connection Command section is visible
    cy.contains("label", "Connection Command").should("be.visible");

    // The command contains the required flags
    cy.get("pre.font-mono")
      .first()
      .should("be.visible")
      .and("contain", "--ca-fingerprint")
      .and("contain", "--qpi-addr")
      .and("contain", `--name "${qpuName}"`)
      .and("contain", '--executor "qiskit_aer"');

    // Systemd Command section is visible
    cy.contains("label", "Installation Command (Systemd)").should("be.visible");
    cy.get("pre.font-mono")
      .last()
      .should("be.visible")
      .and("contain", "install-systemd.sh")
      .and("contain", `QPU_NAME="${qpuName}"`)
      .and("contain", `QPI_DRIVER_VERSION=`)
      .and("contain", 'EXECUTOR="qiskit_aer"');
  });

  it("copies the access token when the copy button is clicked", () => {
    cy.contains("button", "Register QPU").click();

    cy.get('input[placeholder="rigetti-aspen-9"]').type(`copy-tok-${Date.now()}`);
    cy.get("select").select("mock");
    cy.contains("button", "Register Unit").click();

    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");

    // The copy button for the token shows the Copy icon initially
    cy.get('button[title="Copy Token"]')
      .find("svg")
      .should("have.class", "lucide-copy");

    // Click the copy button
    cy.get('button[title="Copy Token"]').click();

    // The icon changes to a checkmark
    cy.get('button[title="Copy Token"]')
      .find("svg")
      .should("have.class", "lucide-check");
  });

  it("copies the connection command when the copy button is clicked", () => {
    cy.contains("button", "Register QPU").click();

    cy.get('input[placeholder="rigetti-aspen-9"]').type(`copy-cmd-${Date.now()}`);
    cy.get("select").select("quantify");
    cy.contains("button", "Register Unit").click();

    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");

    // The copy button for the command shows the Copy icon initially
    cy.get('button[title="Copy Connection Command"]')
      .find("svg")
      .should("have.class", "lucide-copy");

    // Click the copy button
    cy.get('button[title="Copy Connection Command"]').click();

    // The icon changes to a checkmark
    cy.get('button[title="Copy Connection Command"]')
      .find("svg")
      .should("have.class", "lucide-check");

    // The copy button for the systemd command shows the Copy icon initially
    cy.get('button[title="Copy Systemd Command"]')
      .find("svg")
      .should("have.class", "lucide-copy");

    // Click the copy button
    cy.get('button[title="Copy Systemd Command"]').click();

    // The icon changes to a checkmark
    cy.get('button[title="Copy Systemd Command"]')
      .find("svg")
      .should("have.class", "lucide-check");
  });

  it("closes the success screen and shows the new QPU in the grid", () => {
    cy.contains("button", "Register QPU").click();

    const qpuName = `grid-test-${Date.now()}`;

    cy.get('input[placeholder="rigetti-aspen-9"]').type(qpuName);
    cy.get("select").select("mock");
    cy.contains("button", "Register Unit").click();

    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");
    cy.contains("button", "Done").click();

    // Modal is closed
    cy.contains("h3", "QPU Registered Successfully!").should("not.exist");

    // The new QPU appears in the grid
    cy.contains("h3", qpuName).should("exist");
  });
});
