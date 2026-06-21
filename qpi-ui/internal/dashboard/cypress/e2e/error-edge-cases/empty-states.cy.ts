describe("Error & Edge Cases — Empty States", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("shows empty state in recent jobs table when no jobs exist", () => {
    cy.get('input[type="text"]').clear().type("emptyuser@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // Overview tab shows the recent jobs table
    cy.contains("h3", "Recent Job Executions")
      .parent()
      .within(() => {
        cy.contains("No jobs submitted yet.").should("be.visible");
      });
  });

  it("shows empty state in job results panel when no job is selected", () => {
    cy.get('input[type="text"]').clear().type("emptyuser@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Jobs").click();
    cy.contains("h1", "Jobs Console").should("be.visible");

    // Results panel shows the empty-state message
    cy.contains("Select or run a job").should("exist");
    cy.contains("No active job selected").should("exist");
  });

  it("loads QPU Registry without error even when no QPUs exist", () => {
    cy.get('input[type="text"]').clear().type("emptyuser@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "QPU Registry").click();
    cy.contains("h1", "QPU Registry").should("be.visible");

    // The page renders; no crash. Grid may be empty.
    cy.contains("Manage and monitor physical/simulator processing units.").should(
      "be.visible",
    );
  });
});
