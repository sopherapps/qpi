describe("Settings & Request Time — Request Time Validation", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Request Time").click();
    cy.contains("h3", "Request QPU Time").should("be.visible");
  });

  it("does not submit with an empty reason", () => {
    cy.get('input[type="number"]').type("{selectall}{backspace}" + "100");
    cy.get("textarea").clear();
    cy.contains("button", "Submit Time Request").click();

    // Modal should still be open because textarea is required
    cy.contains("h3", "Request QPU Time").should("be.visible");
  });

  it("does not submit with empty seconds", () => {
    cy.get('input[type="number"]').clear();
    cy.get("textarea").clear().type("Valid reason");
    cy.contains("button", "Submit Time Request").click();

    // Modal should still be open because number input is required
    cy.contains("h3", "Request QPU Time").should("be.visible");
  });
});
