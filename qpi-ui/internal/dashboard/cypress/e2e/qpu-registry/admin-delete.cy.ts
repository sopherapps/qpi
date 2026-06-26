describe("QPU Registry — Admin: Delete QPU", () => {
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

  it("shows the Delete button on QPU cards", () => {
    cy.contains("button", "Delete").should("be.visible");
  });

  it("can open and cancel the deletion modal", () => {
    // Click Delete on the first QPU
    cy.contains("button", "Delete").first().click();

    // Modal should appear
    cy.contains("h3", "Delete QPU").should("be.visible");
    cy.contains("Are you sure you want to delete").should("be.visible");

    // Cancel deletion
    cy.contains("button", "Cancel").click();

    // Modal should disappear
    cy.contains("h3", "Delete QPU").should("not.exist");
  });

  it("can delete a QPU successfully", () => {
    // 1. Create a specific QPU to delete
    const qpuToDelete = `delete-test-qpu-${Date.now()}`;
    
    // Register it
    cy.contains("button", "Register QPU").click();
    cy.get('input[placeholder="rigetti-aspen-9"]').type(qpuToDelete);
    cy.get("select").select("mock");
    cy.contains("button", "Register Unit").click();
    cy.contains("h3", "QPU Registered Successfully!").should("be.visible");
    cy.get('svg.lucide-x').parent('button').click();

    // Find the newly created QPU card by its name, and click its Delete button
    cy.contains("h3", qpuToDelete)
      .parents("div.bg-white") // find the card container
      .contains("button", "Delete")
      .click();

    // Confirm deletion in modal
    cy.contains("h3", "Delete QPU").should("be.visible");
    cy.contains("button", "Delete QPU").click();

    // Modal should disappear
    cy.contains("h3", "Delete QPU").should("not.exist");

    // The QPU card should no longer exist
    cy.contains("h3", qpuToDelete).should("not.exist");
  });
});
