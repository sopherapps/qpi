describe("Driver Registry — Admin: Register Driver", () => {
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

    // Navigate to the Drivers page
    cy.contains("button", "Drivers").click();
    cy.contains("h1", "Drivers").should("be.visible");
  });

  it("opens the registration modal and shows the form", () => {
    cy.contains("button", "Register Driver").click();

    cy.contains("h3", "Register Driver").should("be.visible");
    cy.get('input[placeholder="cryostat-monitor-1"]').should("be.visible");
    cy.get('[data-testid="driver-qpu-select"]').should("be.visible");
    cy.get('[data-testid="driver-kind-select"]').should("be.visible");
    cy.get('[data-testid="driver-language-select"]').should("be.visible");
    cy.contains("button", "Register Driver").should("exist");
  });

  it("registers a driver against the seeded QPU and shows the success screen", () => {
    cy.contains("button", "Register Driver").click();

    const driverName = `cypress-test-driver-${Date.now()}`;

    cy.get('input[placeholder="cryostat-monitor-1"]').type(driverName);
    cy.get('[data-testid="driver-qpu-select"]').select("qpu_sim_01");
    cy.get('[data-testid="driver-kind-select"]').select("mock");
    cy.get('[data-testid="driver-language-select"]').select("python");
    cy.get('form').contains("button", "Register Driver").click();

    cy.contains("h3", "Driver Registered Successfully!").should("be.visible");
    cy.contains(`Driver ${driverName}`).should("be.visible");

    // The one-time token is shown and non-empty
    cy.contains("label", "One-Time Token").should("be.visible");
    cy.get("span.font-mono").contains(/\S+/).should("be.visible");

    // The official mock/python build ships systemd + manual CLI + install-and-run snippets.
    // Systemd passes the token via the QPI_TOKEN env var; the others use --token.
    cy.contains("label", "Installation Command (Systemd)").should(
      "be.visible",
    );
    cy.contains("label", "Manual CLI Command").should("be.visible");
    cy.get("pre.font-mono").eq(0).should("contain", "QPI_TOKEN=");
    cy.get("pre.font-mono").eq(1).should("contain", "--token");

    cy.get("svg.lucide-x").parent("button").click();
    cy.contains("h3", driverName).should("be.visible");
  });

  it("requires at least one event for a custom driver", () => {
    cy.contains("button", "Register Driver").click();

    cy.get('input[placeholder="cryostat-monitor-1"]').type(
      `custom-driver-${Date.now()}`,
    );
    cy.get('[data-testid="driver-qpu-select"]').select("qpu_sim_01");
    cy.get('[data-testid="driver-kind-select"]').select("custom");
    cy.get('[data-testid="driver-language-select"]').select("go");

    cy.get('form').contains("button", "Register Driver").click();

    cy.contains("Select at least one event").should("be.visible");
  });

  it("shows install + stub snippets for a custom driver", () => {
    cy.contains("button", "Register Driver").click();

    const driverName = `custom-driver-${Date.now()}`;
    cy.get('input[placeholder="cryostat-monitor-1"]').type(driverName);
    cy.get('[data-testid="driver-qpu-select"]').select("qpu_sim_01");
    cy.get('[data-testid="driver-kind-select"]').select("custom");
    cy.get('[data-testid="driver-language-select"]').select("go");
    cy.contains("JobDispatch").click();

    cy.get('form').contains("button", "Register Driver").click();

    cy.contains("h3", "Driver Registered Successfully!").should("be.visible");
    cy.contains("label", "Install Command").should("be.visible");
    cy.contains("label", "Driver Stub").should("be.visible");
  });
});
