describe("Jobs Console — Default Form State", () => {
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

  it("pre-selects the first online QPU in the dropdown", () => {
    cy.get("select")
      .should("exist")
      .and("not.have.value", "")
      .and("not.have.value", "No online QPUs available");

    // The selected option should show a QPU name, not the empty placeholder
    cy.get("select option:selected").should("not.have.value", "");
  });

  it("shows the Bell state example in the QASM textarea", () => {
    cy.get("textarea")
      .should("exist")
      .and("contain", 'OPENQASM 3.0')
      .and("contain", 'h q[0]')
      .and("contain", 'cx q[0], q[1]')
      .and("contain", 'c = measure q');
  });

  it("defaults shots to 1000", () => {
    cy.get('input[type="number"]')
      .should("exist")
      .and("have.value", "1000");
  });

  it("defaults meas level to 2 (Counts)", () => {
    cy.contains("span", "2 (Counts)").should("exist");

    // The range input should have value 2
    cy.get('input[type="range"]').should("have.value", "2");
  });

  it("shows the Execute Job button enabled when a QPU is selected", () => {
    cy.contains("button", "Execute Job")
      .should("exist")
      .and("not.be.disabled");
  });
});
