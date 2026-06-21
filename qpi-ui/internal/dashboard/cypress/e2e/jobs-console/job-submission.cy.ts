describe("Jobs Console — Job Submission & Results", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Jobs Console").click();
    cy.contains("h1", "Jobs Console").should("be.visible");
  });

  it("submits a job and shows completed status with duration", () => {
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

  it("shows the correct target QPU name in the results panel", () => {
    // Capture the selected QPU name from the dropdown
    cy.get("select option:selected")
      .invoke("text")
      .then((qpuName) => {
        // Execute job
        cy.contains("button", "Execute Job").click();

        // Wait for completion
        cy.contains("div", "completed", { timeout: 15000 }).should(
          "be.visible",
        );

        // The results panel should show the QPU name
        cy.contains("p", `Executed on ${qpuName.trim()}`).should("be.visible");
      });
  });

  it("shows job ID prefix in the results panel after submission", () => {
    cy.contains("button", "Execute Job").click();

    // The results panel should show a job ID (starts with #)
    cy.get("h3")
      .contains(/^#[a-zA-Z0-9]+/)
      .should("be.visible");
  });

  it("shows the counts histogram tab by default for completed jobs", () => {
    cy.contains("button", "Execute Job").click();

    cy.contains("div", "completed", { timeout: 15000 }).should("be.visible");

    // The Counts Histogram tab should be active
    cy.contains("button", "Counts Histogram")
      .should("be.visible")
      .and("have.class", "border-b-2")
      .and("have.class", "border-white");
  });
});
