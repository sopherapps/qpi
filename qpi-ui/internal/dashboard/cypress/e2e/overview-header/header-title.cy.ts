describe("Overview & Header — Page Title", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it('shows "Overview" as the header title on the default tab', () => {
    cy.get("header h2").should("contain", "Overview");
  });

  it('shows "QPU Registry" when on the QPUs tab', () => {
    cy.contains("button", "QPU Registry").click();
    cy.get("header h2").invoke("text").should("match", /qpu registry/i);
  });

  it('shows "Jobs Console" when on the Jobs tab', () => {
    cy.contains("button", "Jobs Console").click();
    cy.get("header h2").invoke("text").should("match", /jobs console/i);
  });

  it('shows "Bookings Overview" when on the Bookings tab', () => {
    cy.contains("button", "Bookings").click();
    cy.get("header h2").invoke("text").should("match", /bookings overview/i);
  });

  it('shows "Settings Overview" when on the Settings tab', () => {
    cy.contains("button", "Profile Settings").click();
    cy.get("header h2").invoke("text").should("match", /settings overview/i);
  });
});
