describe("Jobs Console — Empty State", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("emptyuser@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Jobs").click();
    cy.contains("h1", "Jobs Console").should("be.visible");
  });

  it("shows a helpful empty state before any job is selected", () => {
    // The results panel heading should prompt the user to select or run a job
    cy.contains("h3", "Select or run a job").should("exist");

    // The subtext should indicate no job is active
    cy.contains("p", "No active job selected").should("exist");
  });

  it("does not show job metadata when no job is selected", () => {
    // Duration label should not be visible
    cy.contains("span", "Duration").should("not.exist");

    // Created label should not be visible
    cy.contains("span", "Created").should("not.exist");

    // Status badge should not be visible (match exact text/badge container)
    cy.get(".capitalize").contains("completed").should("not.exist");
    cy.get(".capitalize").contains("pending").should("not.exist");
    cy.get(".capitalize").contains("running").should("not.exist");
  });

  it("shows the visualization placeholder when no job is selected", () => {
    cy.contains(
      "div",
      "Select a completed job to view results visualization.",
    ).should("exist");
  });
});
