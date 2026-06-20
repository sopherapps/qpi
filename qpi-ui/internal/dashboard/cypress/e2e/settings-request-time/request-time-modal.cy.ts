describe("Settings & Request Time — Request Time Modal", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it("opens the Request Time modal from the sidebar", () => {
    cy.contains("button", "Request Time").click();

    cy.contains("h3", "Request QPU Time").should("be.visible");
    cy.get('input[type="number"]').should("be.visible");
    cy.get("textarea").should("be.visible");
    cy.contains("button", "Submit Time Request").should("be.visible");
  });

  it("submits a time request and shows success alert", () => {
    cy.on("window:alert", (txt) => {
      expect(txt).to.include("submitted successfully");
    });

    cy.contains("button", "Request Time").click();
    cy.contains("h3", "Request QPU Time").should("be.visible");

    cy.get('input[type="number"]').clear().type("500");
    cy.get("textarea").type("Need extra time for VQE experiments");
    cy.contains("button", "Submit Time Request").click();

    // Modal closes after successful submission
    cy.contains("h3", "Request QPU Time").should("not.exist");
  });
});
