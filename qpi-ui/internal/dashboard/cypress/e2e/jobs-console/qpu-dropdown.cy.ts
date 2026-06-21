describe("Jobs Console — QPU Dropdown", () => {
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

  it("shows only online QPUs in the dropdown", () => {
    // The dropdown should have at least one option and none should say "offline"
    cy.get("select option").each(($option) => {
      cy.wrap($option).should("not.contain", "offline");
    });
  });

  it("disables Execute Job when no QPU is selected", () => {
    // This is hard to trigger in practice since a QPU is auto-selected,
    // but we can verify the button is disabled when the select has no value
    cy.get("select").invoke("val", "").trigger("change");

    cy.contains("button", "Execute Job").should("be.disabled");
  });

  it("reflects the selected QPU in the submitted job", () => {
    // Get the initially selected QPU name
    cy.get("select option:selected")
      .invoke("text")
      .then((initialQpuName) => {
        const trimmedName = initialQpuName.trim();

        // Submit job
        cy.contains("button", "Execute Job").click();
        cy.contains("div", "completed", { timeout: 15000 }).should(
          "be.visible",
        );

        // Verify the results panel shows the same QPU
        cy.contains("p", `Executed on ${trimmedName}`).should("be.visible");
      });
  });
});
