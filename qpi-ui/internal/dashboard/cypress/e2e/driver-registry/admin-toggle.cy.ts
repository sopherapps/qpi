describe("Driver Registry — Admin: Toggle & Delete Driver", () => {
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

    // Register a driver to toggle/delete
    cy.contains("button", "Register Driver").click();
    const driverName = `toggle-test-driver-${Date.now()}`;
    cy.wrap(driverName).as("driverName");
    cy.get('input[placeholder="cryostat-monitor-1"]').type(driverName);
    cy.get('[data-testid="driver-qpu-select"]').select("qpu_sim_01");
    cy.get('[data-testid="driver-kind-select"]').select("mock");
    cy.get('[data-testid="driver-language-select"]').select("python");
    cy.get("form").contains("button", "Register Driver").click();
    cy.contains("h3", "Driver Registered Successfully!").should("be.visible");
    cy.get("svg.lucide-x").parent("button").click();
  });

  it("shows the new driver as offline until it connects", () => {
    cy.get("@driverName").then((driverName) => {
      cy.contains("h3", driverName as unknown as string)
        .parents("div.bg-white")
        .find('[data-testid="driver-status"]')
        .should("contain", "offline");
    });
  });

  it("toggles a driver from Enabled to Disabled and back", () => {
    cy.get("@driverName").then((driverName) => {
      const card = () =>
        cy.contains("h3", driverName as unknown as string).parents("div.bg-white");

      card().contains("button", "Enabled").click();
      card().contains("button", "Disabled").should("be.visible");
      card().contains("button", "Disabled").click();
      card().contains("button", "Enabled").should("be.visible");
    });
  });

  it("deletes a driver", () => {
    cy.get("@driverName").then((driverName) => {
      const name = driverName as unknown as string;
      cy.contains("h3", name)
        .parents("div.bg-white")
        .contains("button", "Delete")
        .click();

      cy.contains("h3", "Delete Driver").should("be.visible");
      cy.contains("button", "Delete Driver").click();
      cy.contains("h3", "Delete Driver").should("not.exist");
      cy.contains("h3", name).should("not.exist");
    });
  });
});
