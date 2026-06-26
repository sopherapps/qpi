describe("Jobs Console — Admin Job Submission", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    // Login as admin
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Jobs Console").click();
    cy.contains("h1", "Jobs Console").should("be.visible");
  });

  it("submits a job successfully as an admin", () => {
    // Execute job with default form values
    cy.contains("button", "Execute Job").click();

    // Wait for the job to complete
    cy.contains("div", "completed", { timeout: 15000 }).should("be.visible");

    // Verify duration is displayed
    cy.contains("span", "Duration")
      .parent()
      .find("span.font-mono")
      .should("be.visible")
      .and("contain", "s");
  });
});
