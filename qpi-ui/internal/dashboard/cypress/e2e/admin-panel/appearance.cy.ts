describe("Admin Panel — Appearance", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    // Log in as administrator
    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();

    cy.contains("h1", "QPI Interface").should("be.visible");
    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    
    // Navigate to Appearance tab
    cy.contains("button", "Appearance").click();
  });

  it("can create, preview, activate, and delete a theme", () => {
    const themeName = `Cypress Theme ${Date.now()}`;
    const siteName = `Cypress Site ${Date.now()}`;
    const secondThemeName = `Second Theme ${Date.now()}`;

    // 1. Create a new theme
    cy.contains("button", "Create New Theme").click();
    cy.contains("h2", "Create New Theme").should("be.visible");
    
    // Fill out the form
    cy.contains("label", "Theme Name").next("input").type(themeName);
    cy.contains("label", "Site Name").next("input").clear().type(siteName);
    cy.contains("label", "Tagline").next("input").clear().type("Testing 123");
    
    // Make the theme drastically different by injecting Custom CSS
    cy.contains("label", "Custom CSS").next("textarea").clear().type("body { background-color: rgb(255, 0, 0) !important; }", { parseSpecialCharSequences: false, delay: 0 });
    
    // Test Preview functionality (applies CSS tokens temporarily)
    cy.contains("button", /^Preview$/).click().then(() => {
        // Manually run handlePreview code in Cypress just to see if the DOM accepts it!
        // No, I want to see if the React button actually ran it.
    });
    
    // Let's add a small wait just in case React is batching or something, even though Cypress is supposed to handle it.
    cy.wait(500);
    
    // Debug: Assert the style tag exists and has the correct content
    cy.get("#qpi-theme-css-preview").should("exist").invoke("text").then((text) => {
        expect(text).to.include("rgb(255, 0, 0)");
    });
    
    // Reset Preview
    cy.contains("button", "Reset Preview").click();
    
    // Save Theme
    cy.contains("button", "Save Theme").click();
    cy.contains("h2", "Create New Theme").should("not.exist");
    
    // 2. Verify theme is in the table
    cy.contains("td", themeName).should("be.visible");
    
    // 3. Activate the theme
    cy.on("window:confirm", () => true);
    cy.contains("tr", themeName).contains("button", "Activate").click();
    
    // Wait for the background theme application to complete.
    cy.wait(1000);
    
    // Verify the theme was actually applied globally via the style tag
    cy.get("#qpi-theme-css").should("exist").invoke("text").then((text) => {
        expect(text).to.include("rgb(255, 0, 0)");
    });
    
    // 4. Create a second theme to deactivate the first one
    cy.contains("button", "Create New Theme").click();
    cy.contains("label", "Theme Name").next("input").type(secondThemeName);
    cy.contains("label", "Custom CSS").next("textarea").clear().type("body { background-color: rgb(0, 255, 0) !important; }", { parseSpecialCharSequences: false });
    cy.contains("button", "Save Theme").click();
    
    // Verify the second theme is NOT active yet
    cy.contains("tr", secondThemeName).contains("button", "Activate").should("be.visible");
    
    // Activate the second theme so we can delete the first one
    cy.contains("tr", secondThemeName).contains("button", "Activate").click();
    
    // Wait for the UI to update: the second theme becomes ACTIVE
    cy.contains("tr", secondThemeName).contains("ACTIVE").should("be.visible");
    cy.wait(1000); // Wait for the global theme to apply via fetch
    
    // Verify the global theme updated immediately to green via the style tag
    cy.get("#qpi-theme-css").should("exist").invoke("text").then((text) => {
        expect(text).to.include("rgb(0, 255, 0)");
    });
    
    // Assert the first theme is NO LONGER ACTIVE and delete it
    cy.contains("tr", themeName).within(() => {
      cy.contains("ACTIVE").should("not.exist");
      cy.contains("button", "Delete").should("be.visible").click();
    });
    
    // Verify deletion
    cy.contains("td", themeName).should("not.exist");
    
    // Clean up second theme
    cy.contains("tr", secondThemeName).contains("button", "Delete").click();
  });
});
