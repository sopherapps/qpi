describe("Error & Edge Cases — Unauthorized Access", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");
  });

  it("shows 'Access Denied' when a regular user navigates to /#admin", () => {
    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.visit("/#admin");
    cy.contains("Access Denied").should("be.visible");

    // Admin Panel button should not appear in the sidebar
    cy.contains("button", "Admin Panel").should("not.exist");
  });
});
