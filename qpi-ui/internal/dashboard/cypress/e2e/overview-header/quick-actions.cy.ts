describe("Overview & Header — Quick Action Buttons", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it('navigates to Bookings when "Book Slot" is clicked', () => {
    cy.contains("button", "Book Slot").click();
    cy.contains("h1", "Bookings").should("be.visible");
    cy.url().should("include", "#bookings");
  });

  it('navigates to Jobs Console when "Submit Job" is clicked', () => {
    cy.contains("button", "Submit Job").click();
    cy.contains("h1", "Jobs Console").should("be.visible");
    cy.url().should("include", "#jobs");
  });
});
