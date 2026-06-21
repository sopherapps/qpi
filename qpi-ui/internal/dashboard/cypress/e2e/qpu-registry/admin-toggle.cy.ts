describe("QPU Registry — Admin: Toggle QPU", () => {
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

  it("shows the Driver Enable Control toggle on QPU cards", () => {
    cy.contains("span", "Driver Enable Control").should("be.visible");
  });

  it("toggles a QPU from Online to Offline and back", () => {
    // Find a QPU that is currently Online (Enabled)
    cy.contains("button", "Online (Enabled)")
      .should("be.visible")
      .first()
      .as("toggleBtn");

    // Click to toggle Offline
    cy.get("@toggleBtn").click();

    // Verify it becomes Offline (Disabled)
    cy.contains("button", "Offline (Disabled)")
      .should("be.visible")
      .first()
      .as("toggleBtnOffline");

    // Click to toggle back Online
    cy.get("@toggleBtnOffline").click();

    // Verify it becomes Online (Enabled) again
    cy.contains("button", "Online (Enabled)").should("be.visible");
  });

  it("shows the correct styling for enabled vs disabled states", () => {
    // Enabled state: green styling
    cy.contains("button", "Online (Enabled)")
      .first()
      .should("have.class", "text-green-400")
      .and("have.class", "border-green-500/20");

    // Toggle to disabled
    cy.contains("button", "Online (Enabled)").first().click();

    // Disabled state: red styling
    cy.contains("button", "Offline (Disabled)")
      .first()
      .should("have.class", "text-red-400")
      .and("have.class", "border-red-500/20");
  });
});
