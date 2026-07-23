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
    cy.contains("button", "Register Unit").should("be.visible");
  });

  it("registers a QPU and shows the new QPU in the grid", () => {
    cy.contains("button", "Register QPU").click();

    const qpuName = `cypress-test-qpu-${Date.now()}`;

    cy.get('input[placeholder="rigetti-aspen-9"]').type(qpuName);
    cy.contains("button", "Register Unit").click();

    // Modal is closed
    cy.contains("h3", "Register QPU").should("not.exist");

    // The new QPU appears in the grid
    cy.contains("h3", qpuName).should("exist");
  });
});
