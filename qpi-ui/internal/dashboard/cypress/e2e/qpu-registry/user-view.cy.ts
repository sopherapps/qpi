describe("QPU Registry — Regular User View", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    // Log in as regular user
    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // Navigate to QPU Registry
    cy.contains("button", "QPU Registry").click();
    cy.contains("h1", "QPU Registry").should("be.visible");
  });

  it("does NOT show the 'Register QPU' button", () => {
    cy.contains("button", "Register QPU").should("not.exist");
  });

  it("does NOT show the Driver Enable Control toggle", () => {
    cy.contains("span", "Driver Enable Control").should("not.exist");
    cy.contains("button", "Online (Enabled)").should("not.exist");
    cy.contains("button", "Offline (Disabled)").should("not.exist");
  });

  it("shows QPU cards with status and executor info", () => {
    // QPU cards should be visible with basic info
    cy.get("h3").should("have.length.at.least", 1);

    // Each card shows status badge
    cy.contains("span", "online").should("be.visible");

    // Each card shows executor driver
    cy.contains("span", "Executor Driver").should("be.visible");
  });
});
